import os

source_file = "/home/nemetg/projects/datagrid/datagrid.go"
with open(source_file, "r") as f:
    lines = f.readlines()

def extract(ranges):
    out = []
    for start, end in ranges:
        out.extend(lines[start-1:end])
    return "".join(out)

print("Keeping ONLY datagrid.go main body")

keep_ranges = [
    (1, 24),    # Imports and embed FS
    (172, 498), # Handler definitions and core structs
    (937, 1238), # Main execute endpoints and parsing routines
    (1260, 1260), # Keep space
    (1374, 1374), # Keep space
    (1501, 1501), # Keep space
    (1607, 1607), # Keep space
    (1713, 1738), # Pivot engine and generic setup routine
]

kept_code = extract(keep_ranges)
with open("/home/nemetg/projects/datagrid/datagrid.go", "w") as f:
    f.write(kept_code)

print("Pruning complete!")
