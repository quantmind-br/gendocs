# Optimize file scanning with selective hashing and parallel processing

## Overview

The cache.ScanFiles() function calculates SHA256 hashes for every file sequentially, which becomes a bottleneck for large repositories. Many files haven't changed and don't need re-hashing.

## Rationale

ScanFiles walks the entire repository and computes SHA256 for every file using HashFile, which reads complete file contents through io.Copy. For a typical project with 1000+ files averaging 50KB each, this means reading 50MB+ on every analysis. Hashing is CPU-bound and the file walking is I/O bound - both can be parallelized. Additionally, we can skip hashing files whose mtime hasn't changed since the last scan.

---
*This spec was created from ideation and is pending detailed specification.*
