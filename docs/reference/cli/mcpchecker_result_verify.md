## mcpchecker result verify

Verify evaluation results meet thresholds

### Synopsis

Verify that evaluation results meet minimum pass rate thresholds.
Useful as a CI gate to enforce quality standards.

Exits with code 0 if all thresholds are met, code 1 otherwise.
Use 'mcpchecker result summary' to view detailed results.

```
mcpchecker result verify <results-file> [flags]
```

### Options

```
      --assertion float   Minimum assertion pass rate (0.0-1.0)
  -h, --help              help for verify
      --task float        Minimum task pass rate (0.0-1.0)
```

### SEE ALSO

* [mcpchecker result](mcpchecker_result.md)	 - Commands for inspecting and analyzing evaluation result files

