# Test data: R reference

Copied unchanged from `github.com/mioluigi/go-trimmed-mean-demo` at commit `835c248`, where
`r/capture-reference.R` generated them on 2026-11-22 (R 4.4.2, aarch64-apple-darwin, seed 9431; see
`r-session.txt`). Synthetic data; no personal information.

| File | Content |
|---|---|
| `integers-{100,101,1000}.csv`, `floats-{100,101,1000}.csv` | the samples: header `value`, one number per line (floats with 4 decimals) |
| `floats-*-as-read-by-r.txt` | the doubles R read from each float sample, at 17 digits, so a test proves Go reads the same doubles |
| `trimmed-means.tsv` | R's `mean(x, trim)` for 8 symmetric proportions, and our asymmetric definition evaluated in R for 6 pairs, per sample; `full_sort_value` is R's mean of the fully sorted kept values |
