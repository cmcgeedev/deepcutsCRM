#!/usr/bin/env python3
"""Print the first fenced code block that follows a line mentioning `<needle>` in backticks."""
import sys

plan, needle = sys.argv[1], sys.argv[2]
lines = open(plan, encoding="utf-8").read().split("\n")
i = next(k for k, l in enumerate(lines) if f"`{needle}`" in l)
start = next(k for k in range(i, len(lines)) if lines[k].startswith("```")) + 1
end = next(k for k in range(start, len(lines)) if lines[k].startswith("```"))
print("\n".join(lines[start:end]))
