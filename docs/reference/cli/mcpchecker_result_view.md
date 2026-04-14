## mcpchecker result view

Pretty-print evaluation results from a JSON file

### Synopsis

Render the JSON output produced by "mcpchecker check" in a human-friendly format.

Examples:
  mcpchecker result view mcpchecker-netedge-selector-mismatch-out.json
  mcpchecker result view --task netedge-selector-mismatch --max-events 15 results.json

```
mcpchecker result view <results-file> [flags]
```

### Options

```
  -h, --help                   help for view
      --max-events int         Maximum number of timeline entries (thought/command/tool/etc.) to display (0 = unlimited) (default 40)
      --max-line-length int    Maximum characters per line when formatting timeline output (default 100)
      --max-output-lines int   Maximum lines to display for command output in the timeline (default 6)
      --task string            Only show results for tasks whose name contains this value
      --timeline               Include a condensed agent timeline derived from taskOutput (default true)
```

### SEE ALSO

* [mcpchecker result](mcpchecker_result.md)	 - Commands for inspecting and analyzing evaluation result files

