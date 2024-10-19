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
	Run: RunFileServer,
}

func init() {
	fileServerCmd.Flags().IntP("port", "p", 8080, "file server port")
	fileServerCmd.Flags().StringP("dir", "d", ".", "file server directory")
}

func RunFileServer(cmd *cobra.Command, args []string) {
	sharedDirectory, _ := cmd.Flags().GetString("dir")
	port, _ := cmd.Flags().GetInt("port")

	absPath, err := filepath.Abs(sharedDirectory)
	if err != nil {
		fmt.Printf("%s is not exists: %v\n", sharedDirectory, err)
		return
	}

	fileServer := http.FileServer(http.Dir(absPath))

	http.Handle("/", http.StripPrefix("/", fileServer))

	fmt.Printf("fileserver is running,\n\tdirectory: %s，\n\taddress: http://<your-ip>:%d\n", absPath, port)

	if er := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); er != nil {
		fmt.Printf("server start server: %s\n", er)
	}
}
