package distro

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/pysugar/wheels/net/ipaddr"
	"github.com/spf13/cobra"
)

var fileServerCmd = &cobra.Command{
	Use:   `fileserver [-d .] -p 8080`,
	Short: "Start a File Server",
	Long: `
Start a File Server.

Start file server: netool fileserver --dir=. --port=8088
`,
	Run: func(cmd *cobra.Command, args []string) {
		sharedDirectory, _ := cmd.Flags().GetString("dir")
		port, _ := cmd.Flags().GetInt("port")
		verbose, _ := cmd.Flags().GetBool("verbose")

		RunFileServer(sharedDirectory, port, verbose)
	},
}

func init() {
	fileServerCmd.Flags().IntP("port", "p", 8080, "file server port")
	fileServerCmd.Flags().StringP("dir", "d", ".", "file server directory")
	fileServerCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
}

func RunFileServer(sharedDirectory string, port int, verbose bool) {
	absPath, err := filepath.Abs(sharedDirectory)
	if err != nil {
		fmt.Printf("%s is not exists: %v\n", sharedDirectory, err)
		return
	}

	fileServer := http.FileServer(http.Dir(absPath))

	http.Handle("/", http.StripPrefix("/", noCacheMiddleware(fileServer)))

	addrs, err := ipaddr.GetLocalIPv4Addrs(verbose)
	if err != nil {
		log.Printf("Failed to get local IPv4 addresses: %v\n", err)
	}
	if len(addrs) == 0 {
		addrs = []string{"0.0.0.0"}
	}

	fmt.Printf("fileserver is running,\n\tdirectory:\t%s \n\taddress:\thttp://<%s>:%d\n", absPath, addrs[0], port)
	if er := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); er != nil {
		fmt.Printf("server start server: %s\n", er)
	}
}

func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}
