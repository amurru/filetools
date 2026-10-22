package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDirArg(t *testing.T) {
	if got := dirArg(nil); got != "." {
		t.Fatalf("dirArg(nil) = %q, want .", got)
	}
	if got := dirArg([]string{"/tmp/foo"}); got != "/tmp/foo" {
		t.Fatalf("dirArg = %q, want /tmp/foo", got)
	}
}

func TestGetOutputFormatShortcuts(t *testing.T) {
	cases := []struct {
		json, xml, html bool
		output          string
		want            string
	}{
		{json: true, output: "text", want: "json"},
		{xml: true, output: "text", want: "xml"},
		{html: true, output: "text", want: "html"},
		{output: "json", want: "json"},
		{output: "xml", want: "xml"},
		{output: "html", want: "html"},
		{output: "text", want: "text"},
	}
	for _, c := range cases {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("json", false, "")
		cmd.Flags().Bool("xml", false, "")
		cmd.Flags().Bool("html", false, "")
		_ = cmd.Flags().Set("json", boolStr(c.json))
		_ = cmd.Flags().Set("xml", boolStr(c.xml))
		_ = cmd.Flags().Set("html", boolStr(c.html))
		outputFormat = c.output
		got := getOutputFormat(cmd)
		if string(got) != c.want {
			t.Errorf("json=%v xml=%v html=%v output=%s: got %s want %s", c.json, c.xml, c.html, c.output, got, c.want)
		}
	}
	outputFormat = "text"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
