package distro

import (
	"fmt"
	"net/http"
	"path/filepath"

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

		RunFileServer(sharedDirectory, port)
	},
}

func init() {
	fileServerCmd.Flags().IntP("port", "p", 8080, "file server port")
	fileServerCmd.Flags().StringP("dir", "d", ".", "file server directory")
}

func RunFileServer(sharedDirectory string, port int) {
	absPath, err := filepath.Abs(sharedDirectory)
	if err != nil {
		fmt.Printf("%s is not exists: %v\n", sharedDirectory, err)
		return
	}

	fileServer := http.FileServer(http.Dir(absPath))

	http.Handle("/", http.StripPrefix("/", noCacheMiddleware(fileServer)))

	fmt.Printf("fileserver is running,\n\tdirectory:\t%s \n\taddress:\thttp://<your-ip>:%d\n", absPath, port)

	if er := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); er != nil {
		fmt.Printf("server start server: %s\n", er)
	}
}

func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 HTTP 头部，禁止浏览器缓存
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}
