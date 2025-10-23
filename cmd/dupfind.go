package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// dupfindCmd represents the dupfind command
var dupfindCmd = &cobra.Command{
	Use:   "dupfind",
	Short: "Find duplicate files",
	Long: `Find duplicate files in a directory tree.

This command will find duplicate files in a directory tree. It will
compare the hashes of each file and report any files that are
identical. File names are not important.

The output will be a list of files that are identical. The files will
be listed in the order that they were found, with the first file
listed being the first duplicate. The files will be listed in
alphabetical order by filename.

If the directory is not specified, the current directory will be used.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dupfind called")
	},
}

func init() {
	rootCmd.AddCommand(dupfindCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dupfindCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dupfindCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
