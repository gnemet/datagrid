import json
import jsonschema
import os
import sys

def validate_catalog(schema_path, catalog_path):
    with open(schema_path, 'r') as f:
        schema = json.load(f)
    
    with open(catalog_path, 'r') as f:
        catalog = json.load(f)
    
    try:
        jsonschema.validate(instance=catalog, schema=schema)
        print(f"✅ {os.path.basename(catalog_path)} is valid.")
        return True
    except jsonschema.exceptions.ValidationError as e:
        print(f"❌ {os.path.basename(catalog_path)} is invalid!")
        print(f"   Reason: {e.message}")
        print(f"   Path: {' -> '.join([str(p) for p in e.path])}")
        return False

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python validate.py <schema_path> <catalog_path1> <catalog_path2> ...")
        sys.exit(1)
    
    schema_path = sys.argv[1]
    catalogs = sys.argv[2:]
    
    all_valid = True
    for cat in catalogs:
        if not validate_catalog(schema_path, cat):
            all_valid = False
    
    if not all_valid:
        sys.exit(1)
