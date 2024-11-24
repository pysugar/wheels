package distro

import (
	"fmt"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var devtoolCmd = &cobra.Command{
	Use:   `devtool -p 8080`,
	Short: "Start a DevTool for HTTP",
	Long: `
Start a DevTool for HTTP.

Start a DevTool for HTTP: netool devtool --port=8080 --verbose
`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		verbose, _ := cmd.Flags().GetBool("verbose")
		RunDevtoolHTTPServer(port, verbose)
	},
}

func init() {
	devtoolCmd.Flags().IntP("port", "p", 8080, "http proxy	 port")
	devtoolCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
}

func RunDevtoolHTTPServer(port int, verbose bool) {
	debugHandler := extensions.CORSMiddleware(http.HandlerFunc(extensions.DebugHandler))
	debugHandlerJSON := extensions.CORSMiddleware(http.HandlerFunc(extensions.DebugHandlerJSON))
	if verbose {
		debugHandler = extensions.LoggingMiddleware(debugHandler)
		debugHandlerJSON = extensions.LoggingMiddleware(debugHandlerJSON)
	}
	http.Handle("/", extensions.CORSMiddleware(debugHandler))
	http.Handle("/json", extensions.CORSMiddleware(debugHandlerJSON))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting debug server at http://localhost%s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
