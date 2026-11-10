package cli

import (
	"fmt"
	"io"
	"github.com/FityatulhaqRosyidi/sealix/internal/trustcmd"
	"github.com/FityatulhaqRosyidi/sealix/internal/verifiycmd"
	// "github.com/FityatulhaqRosyidi/sealix/internal/keygencmd"
	// "github.com/FityatulhaqRosyidi/sealix/internal/signcmd"
)

func Run(
	args []string,
	out io.Writer,
	errOut io.Writer,
	version string,
) int {
	if len(args) == 0 {
		help(out)
		return 2
	}
	switch args[0] {
		case "--help", "-h", "help":
			help(out)
			return 0
		case "--version", "version":
			fmt.Fprintln(out, "sealix", version)
			return 0
		case "trust":
			return trustcmd.Run(args[1:], out, errOut)
		case "verify":
			return verifiycmd.Run(args[1:], out, errOut)
		// case "keygen":
		// 	return keygencmd.Run(args[1:], out, errOut)
		// case "sign":
		// 	return signcmd.Run(args[1:], out, errOut)
		default:
			fmt.Fprintln(
				errOut,
				"subcommand tidak dikenali: %s\n",
				args[0],
			)
			help(out)
			return 2
	}
}


func help(w io.Writer) {
	fmt.Fprintln(w, 
		
`sealix - Sistem penandatanganan artefak cicd

Usage: 
		sealix <keygen|sign|trust|verify> [flags]

Commands:

		trust add
			--trust-store PATH
			--public-key PEM
			--package NAME
			[--keyid HEX]

		trust list
			--trust-store PATH
		
		trust remove
			--trust-store PATH
			--keyid HEX

		verify
			--file PATH
			--bundle PATH
			--trust-store PATH
			--package NAME
			--version VERSION
			[--json]
		
		--help        Bantuan
		--version     Versi Aplikasi

`)
}