## mcpchecker result diff

Compare two evaluation results

### Synopsis

Compare evaluation results between two runs (e.g., main vs PR).

Shows regressions, improvements, new tasks, removed tasks, and overall pass rate changes.
Useful for posting on pull requests to show impact of changes.

Example:
  mcpchecker result diff --base results-main.json --current results-pr.json
  mcpchecker result diff --base results-main.json --current results-pr.json --output markdown

```
mcpchecker result diff --base <results-file> --current <results-file> [flags]
```

### Options

```
      --base string      Base results file (e.g., main branch)
      --current string   Current results file (e.g., PR branch)
  -h, --help             help for diff
  -o, --output string    Output format (text, markdown) (default "text")
```

### SEE ALSO

* [mcpchecker result](mcpchecker_result.md)	 - Commands for inspecting and analyzing evaluation result files

