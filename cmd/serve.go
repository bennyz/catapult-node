package cmd

import (
	"github.com/PUMATeam/catapult-node/api"
	"github.com/spf13/cobra"
)

var port int

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start catapult node server",
	Run: func(cmd *cobra.Command, args []string) {
		api.Start(port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 8888, "Port for which to listen")

}
