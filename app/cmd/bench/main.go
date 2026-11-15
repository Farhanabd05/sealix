package main

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FityatulhaqRosyidi/sealix/app/internal/keys"
	"github.com/FityatulhaqRosyidi/sealix/app/internal/manifest"
	"github.com/FityatulhaqRosyidi/sealix/app/internal/sigalg"
)

type artifact struct {
	name string
	data []byte
	path string
}

type rawRow struct {
	metric, profile, artifact string
	iteration                 int
	ns                        int64
}

type summaryRow struct {
	metric, profile, artifact string
	n                         int
	median, p95, min, max     time.Duration
}

type sizeRow struct {
	profile, artifact, kind string
	bytes                   int
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	bin := flag.String("bin", "", "path binary sealix")
	outDir := flag.String("out", "", "folder hasil")
	dataDir := flag.String("data", "", "folder artefak")
	n := flag.Int("n", 30, "jumlah pengukuran")
	warmup := flag.Int("warmup", 5, "jumlah pemanasan")
	keygenN := flag.Int("keygen-n", 10, "jumlah pengukuran keygen")
	sizesArg := flag.String("sizes", "1024,1048576,67108864", "ukuran artefak dalam byte")
	flag.Parse()

	if *bin == "" || *outDir == "" {
		fatalf("-bin dan -out wajib diisi")
	}
	if *n <= 0 || *warmup < 0 || *keygenN <= 0 {
		fatalf("n, warmup, dan keygen-n tidak valid")
	}
	sizes, err := parseSizes(*sizesArg)
	if err != nil {
		fatalf("sizes: %v", err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatalf("membuat folder hasil: %v", err)
	}

	ownedData := false
	if *dataDir == "" {
		*dataDir, err = os.MkdirTemp("", "sealix-bench-data-*")
		if err != nil {
			fatalf("membuat folder data: %v", err)
		}
		ownedData = true
	}
	if ownedData {
		defer os.RemoveAll(*dataDir)
	}
	artifacts, err := prepareArtifacts(*dataDir, sizes)
	if err != nil {
		fatalf("menyiapkan artefak: %v", err)
	}

	var raw []rawRow
	var summaries []summaryRow
	var sizesOut []sizeRow
	profiles := []string{sigalg.ProfileEd25519, sigalg.ProfileECDSA, sigalg.ProfileRSAPSS}
	loadStart := readLoadAverage()

	hashCases := make([]benchCase, len(artifacts))
	for i, a := range artifacts {
		hashCases[i] = benchCase{
			metric: "hash", artifact: a.name,
			fn: func() error {
				_ = sha256.Sum256(a.data)
				return nil
			},
		}
	}
	hashSamples, err := measureInterleaved(hashCases, *n, *warmup)
	if err != nil {
		fatalf("hash: %v", err)
	}
	for i, samples := range hashSamples {
		addResults(&raw, &summaries, "hash", "-", artifacts[i].name, samples)
	}

	for _, profile := range profiles {
		alg, ok := sigalg.Lookup(profile)
		if !ok {
			fatalf("profil tidak dikenal: %s", profile)
		}
		keygenSamples, err := measure(*keygenN, *warmup, func() error {
			_, err := alg.Generate()
			return err
		})
		if err != nil {
			fatalf("keygen %s: %v", profile, err)
		}
		addResults(&raw, &summaries, "keygen", profile, "-", keygenSamples)

		signer, err := alg.Generate()
		if err != nil {
			fatalf("membuat kunci %s: %v", profile, err)
		}
		addKeySizes(&sizesOut, profile, signer)

		signCases := make([]benchCase, 0, len(artifacts))
		verifyCases := make([]benchCase, 0, len(artifacts))
		cliSignCases := make([]benchCase, 0, len(artifacts))
		cliVerifyCases := make([]benchCase, 0, len(artifacts))
		var cliPreparedCases []cliPrepared
		for _, a := range artifacts {
			keyID, err := keys.Fingerprint(signer.Public())
			if err != nil {
				fatalf("fingerprint %s: %v", profile, err)
			}
			digest := sha256.Sum256(a.data)
			m := manifest.Manifest{
				Package: "contoh-app", Version: "1.0.0", Algorithm: profile,
				KeyID: keyID, Artifact: manifest.Artifact{
					Name: filepath.Base(a.path), Size: int64(len(a.data)),
					SHA256: hex.EncodeToString(digest[:]),
				},
			}
			manifestBytes, err := m.Serialize()
			if err != nil {
				fatalf("manifest %s: %v", a.name, err)
			}
			signature, err := alg.Sign(signer, manifestBytes)
			if err != nil {
				fatalf("signature %s/%s: %v", profile, a.name, err)
			}
			publicKey := signer.Public()
			sizesOut = append(sizesOut, sizeRow{profile, a.name, "signature", alg.SigLen()})

			signCases = append(signCases, benchCase{
				metric: "sign_manifest", profile: profile, artifact: a.name,
				fn: func() error {
					_, err := alg.Sign(signer, manifestBytes)
					return err
				},
			})
			verifyCases = append(verifyCases, benchCase{
				metric: "verify_manifest", profile: profile, artifact: a.name,
				fn: func() error {
					if !alg.Verify(publicKey, manifestBytes, signature) {
						return fmt.Errorf("signature tidak valid")
					}
					return nil
				},
			})
			cli, err := prepareCLI(*bin, a, signer)
			if err != nil {
				fatalf("menyiapkan end-to-end %s/%s: %v", profile, a.name, err)
			}
			cliSignCases = append(cliSignCases, benchCase{
				metric: "cli_sign", profile: profile, artifact: a.name, fn: cli.sign,
			})
			cliVerifyCases = append(cliVerifyCases, benchCase{
				metric: "cli_verify", profile: profile, artifact: a.name, fn: cli.verify,
			})
			sizesOut = append(sizesOut, sizeRow{profile, a.name, "bundle", cli.bundleSize})
			cliPreparedCases = append(cliPreparedCases, cli)
		}

		measurements := []struct {
			cases  []benchCase
			metric string
		}{
			{signCases, "sign_manifest"},
			{verifyCases, "verify_manifest"},
			{cliSignCases, "cli_sign"},
			{cliVerifyCases, "cli_verify"},
		}
		for _, measurement := range measurements {
			samples, err := measureInterleaved(measurement.cases, *n, *warmup)
			if err != nil {
				fatalf("%s %s: %v", measurement.metric, profile, err)
			}
			for i, values := range samples {
				addResults(&raw, &summaries, measurement.metric, profile, artifacts[i].name, values)
			}
		}
		for _, cli := range cliPreparedCases {
			cli.cleanup()
		}
	}

	printNoiseWarnings(summaries, profiles, artifacts)
	if err := writeOutputs(*outDir, raw, summaries, sizesOut); err != nil {
		fatalf("menulis hasil: %v", err)
	}
	if err := writeEnvironment(filepath.Join(*outDir, "env.txt"), loadStart); err != nil {
		fatalf("menulis environment: %v", err)
	}
	printSummary(summaries)
}

type cliPrepared struct {
	sign       func() error
	verify     func() error
	bundleSize int
	cleanup    func()
}

func prepareCLI(bin string, a artifact, signer crypto.Signer) (cliPrepared, error) {
	dir, err := os.MkdirTemp("", "sealix-bench-cli-*")
	if err != nil {
		return cliPrepared{}, err
	}
	prefix := filepath.Join(dir, "k")
	if _, err := keys.WriteKeyPair(prefix, signer, true); err != nil {
		os.RemoveAll(dir)
		return cliPrepared{}, err
	}
	trust := filepath.Join(dir, "trust.json")
	if err := runCommand(bin, "trust", "add", "--key", prefix+".pub.pem", "--package", "contoh-app", "--store", trust); err != nil {
		os.RemoveAll(dir)
		return cliPrepared{}, err
	}
	bundlePath := filepath.Join(dir, "bundle.json")
	signArgs := []string{"sign", "--file", a.path, "--package", "contoh-app", "--version", "1.0.0", "--key", prefix + ".key.pem", "--out", bundlePath, "--force"}
	if err := runCommand(bin, signArgs[:len(signArgs)-1]...); err != nil {
		os.RemoveAll(dir)
		return cliPrepared{}, err
	}
	info, err := os.Stat(bundlePath)
	if err != nil {
		os.RemoveAll(dir)
		return cliPrepared{}, err
	}
	return cliPrepared{
		sign: func() error {
			return runCommand(bin, signArgs...)
		},
		verify: func() error {
			return runCommand(bin, "verify", "--file", a.path, "--bundle", bundlePath, "--trust-store", trust, "--package", "contoh-app", "--version", "1.0.0", "--json")
		},
		bundleSize: int(info.Size()),
		cleanup:    func() { _ = os.RemoveAll(dir) },
	}, nil
}

func parseSizes(value string) ([]int, error) {
	var out []int
	for _, item := range strings.Split(value, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("ukuran tidak valid %q", item)
		}
		out = append(out, n)
	}
	return out, nil
}

func prepareArtifacts(dir string, sizes []int) ([]artifact, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewSource(42))
	artifacts := make([]artifact, 0, len(sizes))
	for _, size := range sizes {
		data := make([]byte, size)
		if _, err := rng.Read(data); err != nil {
			return nil, err
		}
		name := fmt.Sprintf("bench-%s.bin", sizeLabel(size))
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact{name: name, data: data, path: path})
	}
	return artifacts, nil
}

func sizeLabel(size int) string {
	switch {
	case size%(1024*1024) == 0:
		return fmt.Sprintf("%dMiB", size/(1024*1024))
	case size%1024 == 0:
		return fmt.Sprintf("%dKiB", size/1024)
	default:
		return strconv.Itoa(size) + "B"
	}
}

func addResults(raw *[]rawRow, summaries *[]summaryRow, metric, profile, artifact string, samples []time.Duration) {
	for i, sample := range samples {
		*raw = append(*raw, rawRow{metric, profile, artifact, i + 1, sample.Nanoseconds()})
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	*summaries = append(*summaries, summaryRow{
		metric: metric, profile: profile, artifact: artifact, n: len(samples),
		median: median(sorted), p95: percentile(sorted, 0.95),
		min: sorted[0], max: sorted[len(sorted)-1],
	})
}

func addKeySizes(out *[]sizeRow, profile string, signer crypto.Signer) {
	privateDER, err := x509.MarshalPKCS8PrivateKey(signer)
	if err != nil {
		fatalf("marshal private key: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		fatalf("marshal public key: %v", err)
	}
	*out = append(*out,
		sizeRow{profile, "-", "private_key_der", len(privateDER)},
		sizeRow{profile, "-", "public_key_der", len(publicDER)},
	)
}

func runCommand(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func writeOutputs(dir string, raw []rawRow, summaries []summaryRow, sizes []sizeRow) error {
	rawFile, err := os.Create(filepath.Join(dir, "raw.csv"))
	if err != nil {
		return err
	}
	rawWriter := csv.NewWriter(rawFile)
	_ = rawWriter.Write([]string{"metrik", "profil", "artefak", "iterasi", "ns"})
	for _, row := range raw {
		_ = rawWriter.Write([]string{row.metric, row.profile, row.artifact, strconv.Itoa(row.iteration), strconv.FormatInt(row.ns, 10)})
	}
	rawWriter.Flush()
	if err := rawWriter.Error(); err != nil {
		rawFile.Close()
		return err
	}
	if err := rawFile.Close(); err != nil {
		return err
	}

	summaryFile, err := os.Create(filepath.Join(dir, "summary.csv"))
	if err != nil {
		return err
	}
	summaryWriter := csv.NewWriter(summaryFile)
	_ = summaryWriter.Write([]string{"metrik", "profil", "artefak", "n", "median_us", "p95_us", "min_us", "max_us"})
	for _, row := range summaries {
		_ = summaryWriter.Write([]string{row.metric, row.profile, row.artifact, strconv.Itoa(row.n), durationMicros(row.median), durationMicros(row.p95), durationMicros(row.min), durationMicros(row.max)})
	}
	summaryWriter.Flush()
	if err := summaryWriter.Error(); err != nil {
		summaryFile.Close()
		return err
	}
	if err := summaryFile.Close(); err != nil {
		return err
	}

	sizeFile, err := os.Create(filepath.Join(dir, "sizes.csv"))
	if err != nil {
		return err
	}
	sizeWriter := csv.NewWriter(sizeFile)
	_ = sizeWriter.Write([]string{"profil", "artefak", "jenis", "bytes"})
	for _, row := range sizes {
		_ = sizeWriter.Write([]string{row.profile, row.artifact, row.kind, strconv.Itoa(row.bytes)})
	}
	sizeWriter.Flush()
	if err := sizeWriter.Error(); err != nil {
		sizeFile.Close()
		return err
	}
	return sizeFile.Close()
}

func durationMicros(value time.Duration) string {
	return fmt.Sprintf("%.3f", float64(value)/float64(time.Microsecond))
}

func writeEnvironment(path, loadStart string) error {
	model := readProcLine("/proc/cpuinfo", "model name")
	mem := readProcLine("/proc/meminfo", "MemTotal")
	version, _ := os.ReadFile("/proc/version")
	lines := []string{
		"runtime=" + runtime.Version(),
		"goos=" + runtime.GOOS,
		"goarch=" + runtime.GOARCH,
		fmt.Sprintf("cpu_count=%d", runtime.NumCPU()),
		"cpu_model=" + model,
		"mem_total=" + mem,
		"proc_version=" + strings.TrimSpace(string(version)),
		"date=" + time.Now().Format(time.RFC3339),
		"loadavg_start=" + loadStart,
		"loadavg_end=" + readLoadAverage(),
	}
	if strings.Contains(strings.ToLower(string(version)), "microsoft") {
		lines = append(lines, "wsl2=true")
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func readLoadAverage() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(strings.SplitN(string(data), " ", 2)[0])
}

func printNoiseWarnings(rows []summaryRow, profiles []string, artifacts []artifact) {
	for _, profile := range profiles {
		for _, metric := range []string{"sign_manifest", "verify_manifest"} {
			var values []time.Duration
			for _, row := range rows {
				if row.profile == profile && row.metric == metric {
					values = append(values, row.median)
				}
			}
			if len(values) != len(artifacts) || len(values) == 0 {
				continue
			}
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			if float64(values[len(values)-1]) > 1.3*float64(values[0]) {
				fmt.Printf("PERINGATAN: derau tinggi pada %s %s (%.2fx)\n",
					metric, profile, float64(values[len(values)-1])/float64(values[0]))
			}
		}
	}
}

func readProcLine(path, prefix string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unavailable"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix+":"))
		}
	}
	return "unavailable"
}

func printSummary(rows []summaryRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].metric != rows[j].metric {
			return rows[i].metric < rows[j].metric
		}
		if rows[i].profile != rows[j].profile {
			return rows[i].profile < rows[j].profile
		}
		return rows[i].artifact < rows[j].artifact
	})
	fmt.Println("| metrik | profil | artefak | n | median_us | p95_us |")
	fmt.Println("|---|---|---|---:|---:|---:|")
	for _, row := range rows {
		fmt.Printf("| %s | %s | %s | %d | %s | %s |\n",
			row.metric, row.profile, row.artifact, row.n,
			durationMicros(row.median), durationMicros(row.p95))
	}
}
