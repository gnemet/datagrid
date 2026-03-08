import os

source_file = "/home/nemetg/projects/datagrid/datagrid.go"
with open(source_file, "r") as f:
    lines = f.readlines()

def extract(ranges):
    out = []
    for start, end in ranges:
        out.extend(lines[start-1:end])
    return "".join(out)

# 1. Models
models_imports = """package datagrid

import (
	"encoding/json"
	"fmt"
	"strings"
)
"""
models_code = extract([(25, 171)])
with open("/home/nemetg/projects/datagrid/datagrid_models.go", "w") as f:
    f.write(models_imports + "\n" + models_code)

# 2. SQL
sql_imports = """package datagrid

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)
"""
sql_code = extract([(1261, 1373), (1375, 1500), (1502, 1606), (1608, 1712)])
with open("/home/nemetg/projects/datagrid/datagrid_sql.go", "w") as f:
    f.write(sql_imports + "\n" + sql_code)

# 3. Export
export_imports = """package datagrid

import (
	"io"
	"encoding/csv"
	"fmt"
	"log/slog"
)
"""
export_code = extract([(1239, 1259)])
with open("/home/nemetg/projects/datagrid/datagrid_export.go", "w") as f:
    f.write(export_imports + "\n" + export_code)

# 4. Template Functions
tpl_imports = """package datagrid

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"reflect"
	"strings"
)
"""
tpl_code = extract([(1739, 1819), (1821, 1831), (1833, 1873), (1875, 1884), (1886, 1890)])
with open("/home/nemetg/projects/datagrid/datagrid_templates.go", "w") as f:
    f.write(tpl_imports + "\n" + tpl_code)

print("Split datagrid execute!")
