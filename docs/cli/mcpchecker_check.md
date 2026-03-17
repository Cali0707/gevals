## mcpchecker check

Run an evaluation

### Synopsis

Run an evaluation using the specified eval configuration file.

```
mcpchecker check [eval-config-file] [flags]
```

### Options

```
  -h, --help                    help for check
  -l, --label-selector string   Filter taskSets by label (format: key=value, e.g., suite=kubernetes)
  -o, --output string           Output format (text, json) (default "text")
  -r, --run string              Regular expression to match task names to run (unanchored, like go test -run)
  -v, --verbose                 Verbose output
```

### SEE ALSO

* [mcpchecker](mcpchecker.md)	 - MCP evaluation framework

