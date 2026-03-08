    (function () {
        'use strict';
        var Pivot2 = {
            toggle: function (row) {
                var key = row.dataset.key;
                var depth = parseInt(row.dataset.depth, 10);
                var chevron = row.querySelector('.pivot2-chevron');
                var table = row.closest('table');
                var rows = Array.from(table.querySelectorAll('tr.pivot2-row'));
                var rowIdx = rows.indexOf(row);

                if (chevron.classList.contains('pivot2-expanded')) {
                    chevron.classList.remove('pivot2-expanded');
                    chevron.classList.add('pivot2-collapsed');
                    for (var i = rowIdx + 1; i < rows.length; i++) {
                        var childDepth = parseInt(rows[i].dataset.depth, 10);
                        if (childDepth <= depth) break;
                        rows[i].classList.add('pivot2-hidden');
                        var cc = rows[i].querySelector('.pivot2-chevron');
                        if (cc) { cc.classList.remove('pivot2-expanded'); cc.classList.add('pivot2-collapsed'); }
                    }
                } else {
                    chevron.classList.remove('pivot2-collapsed');
                    chevron.classList.add('pivot2-expanded');
                    for (var i = rowIdx + 1; i < rows.length; i++) {
                        var childDepth = parseInt(rows[i].dataset.depth, 10);
                        if (childDepth <= depth) break;
                        if (childDepth === depth + 1) rows[i].classList.remove('pivot2-hidden');
                    }
                }
            },
            expandAll: function () {
                var w = document.getElementById('dg-pivot2-wrapper');
                if (!w) return;
                w.querySelectorAll('tr.pivot2-row').forEach(function (r) {
                    r.classList.remove('pivot2-hidden');
                    var c = r.querySelector('.pivot2-chevron');
                    if (c) { c.classList.remove('pivot2-collapsed'); c.classList.add('pivot2-expanded'); }
                });
            },
            collapseAll: function () {
                var w = document.getElementById('dg-pivot2-wrapper');
                if (!w) return;
                var defOpen = parseInt(w.dataset.defaultOpen || "0", 10);
                w.querySelectorAll('tr.pivot2-row').forEach(function (r) {
                    var depth = parseInt(r.dataset.depth, 10);
                    if (depth > defOpen) {
                        r.classList.add('pivot2-hidden');
                    } else {
                        r.classList.remove('pivot2-hidden');
                    }
                    var c = r.querySelector('.pivot2-chevron');
                    if (c) {
                        if (depth >= defOpen) {
                            c.classList.remove('pivot2-expanded');
                            c.classList.add('pivot2-collapsed');
                        } else {
                            c.classList.remove('pivot2-collapsed');
                            c.classList.add('pivot2-expanded');
                        }
                    }
                });
            },
            // Smart filter: supports {column} > value, {col} < val AND {col2} >= val2, or plain text
            localFilter: function (term) {
                var w = document.getElementById('dg-pivot2-wrapper');
                if (!w) return;
                var input = term.trim();
                var rows = w.querySelectorAll('tr.pivot2-row');
                var defOpen = parseInt(w.dataset.defaultOpen || "0", 10);

                if (!input) {
                    // Clear filter: return to default open state
                    rows.forEach(function (r) {
                        var depth = parseInt(r.dataset.depth, 10);
                        r.classList.remove('pivot2-filtered-out');
                        if (depth > defOpen) {
                            r.classList.add('pivot2-hidden');
                        } else {
                            r.classList.remove('pivot2-hidden');
                        }
                        var c = r.querySelector('.pivot2-chevron');
                        if (c) {
                            if (depth >= defOpen) {
                                c.classList.remove('pivot2-expanded'); c.classList.add('pivot2-collapsed');
                            } else {
                                c.classList.remove('pivot2-collapsed'); c.classList.add('pivot2-expanded');
                            }
                        }
                    });
                    return;
                }

                // Try to parse smart expressions: {col} op val [AND {col} op val ...]
                var exprRe = /\{([^}]+)\}\s*(>=|<=|!=|>|<|=)\s*(-?\d+\.?\d*)/g;
                var conditions = [];
                var m;
                while ((m = exprRe.exec(input)) !== null) {
                    conditions.push({ col: m[1].trim(), op: m[2], val: parseFloat(m[3]) });
                }

                if (conditions.length > 0) {
                    this._smartFilter(w, rows, conditions);
                } else {
                    this._textFilter(w, rows, input.toLowerCase());
                }
            },

            // Column-value expression filter
            _smartFilter: function (w, rows, conditions) {
                // Build column index map from thead
                var headers = w.querySelectorAll('table thead th');
                var colMap = {}; // header text → column index (0-based)
                headers.forEach(function (th, idx) {
                    colMap[th.textContent.trim().toLowerCase()] = idx;
                });

                // Resolve column indices for each condition
                var resolved = conditions.map(function (c) {
                    var idx = colMap[c.col.toLowerCase()];
                    return { idx: idx, op: c.op, val: c.val };
                }).filter(function (c) { return c.idx !== undefined; });

                if (resolved.length === 0) return; // no valid columns found

                var rowArr = Array.from(rows);
                // First: hide everything
                rowArr.forEach(function (r) { r.classList.add('pivot2-filtered-out'); r.classList.remove('pivot2-hidden'); });

                // Check depth-0 rows against all conditions
                var ops = {
                    '>': function (a, b) { return a > b }, '<': function (a, b) { return a < b },
                    '>=': function (a, b) { return a >= b }, '<=': function (a, b) { return a <= b },
                    '=': function (a, b) { return a === b }, '!=': function (a, b) { return a !== b }
                };

                rowArr.forEach(function (r, idx) {
                    if (parseInt(r.dataset.depth, 10) !== 0) return;
                    var cells = r.querySelectorAll('td');
                    var match = resolved.every(function (cond) {
                        var cell = cells[cond.idx];
                        if (!cell) return false;
                        var raw = cell.textContent.replace(/[^\d.\-]/g, '').trim();
                        var num = parseFloat(raw);
                        if (isNaN(num)) return false;
                        return ops[cond.op](num, cond.val);
                    });
                    if (match) {
                        // Show this row and all its descendants
                        r.classList.remove('pivot2-filtered-out');
                        var c = r.querySelector('.pivot2-chevron');
                        if (c) { c.classList.remove('pivot2-collapsed'); c.classList.add('pivot2-expanded'); }
                        for (var j = idx + 1; j < rowArr.length; j++) {
                            if (parseInt(rowArr[j].dataset.depth, 10) === 0) break;
                            rowArr[j].classList.remove('pivot2-filtered-out');
                            var cc = rowArr[j].querySelector('.pivot2-chevron');
                            if (cc) { cc.classList.remove('pivot2-collapsed'); cc.classList.add('pivot2-expanded'); }
                        }
                    }
                });
            },

            // Plain text search (original behavior)
            _textFilter: function (w, rows, needle) {
                var rowArr = Array.from(rows);
                rowArr.forEach(function (r) { r.classList.add('pivot2-filtered-out'); r.classList.remove('pivot2-hidden'); });
                rowArr.forEach(function (r, idx) {
                    var label = (r.querySelector('.pivot2-text') || {}).textContent || '';
                    if (label.toLowerCase().indexOf(needle) >= 0) {
                        r.classList.remove('pivot2-filtered-out');
                        var myDepth = parseInt(r.dataset.depth, 10);
                        for (var j = idx - 1; j >= 0; j--) {
                            var anc = rowArr[j];
                            var ancDepth = parseInt(anc.dataset.depth, 10);
                            if (ancDepth < myDepth) {
                                anc.classList.remove('pivot2-filtered-out');
                                var c = anc.querySelector('.pivot2-chevron');
                                if (c) { c.classList.remove('pivot2-collapsed'); c.classList.add('pivot2-expanded'); }
                                myDepth = ancDepth;
                            }
                            if (ancDepth === 0) break;
                        }
                    }
                });
            },

            // ── Column autocomplete ──
            _colNames: null,
            _suggestIdx: -1,

            initAutocomplete: function () {
                var self = this;
                var input = document.querySelector('.pivot2-search');
                var box = document.getElementById('pivot2-col-suggest');
                if (!input || !box) return;

                self._colNames = [];
                var w = document.getElementById('dg-pivot2-wrapper');
                if (w) {
                    var ths = w.querySelectorAll('table thead th');
                    var firstRow = w.querySelector('table tbody tr.pivot2-row');
                    var firstCells = firstRow ? firstRow.querySelectorAll('td') : [];
                    ths.forEach(function (th, idx) {
                        var name = th.textContent.trim();
                        if (!name) return;
                        var type = 'text';
                        if (idx > 0 && firstCells[idx]) {
                            var val = firstCells[idx].textContent.trim().replace(/[,\s]/g, '');
                            if (val && !isNaN(parseFloat(val))) type = 'num';
                        }
                        self._colNames.push({ name: name, type: type });
                    });
                }

                input.addEventListener('keydown', function (e) {
                    if (!box.classList.contains('active')) return;
                    var items = box.querySelectorAll('.pivot2-col-suggest-item');
                    if (e.key === 'ArrowDown') {
                        e.preventDefault();
                        self._suggestIdx = Math.min(self._suggestIdx + 1, items.length - 1);
                        self._hlSuggest(items);
                    } else if (e.key === 'ArrowUp') {
                        e.preventDefault();
                        self._suggestIdx = Math.max(self._suggestIdx - 1, 0);
                        self._hlSuggest(items);
                    } else if (e.key === 'Enter' && self._suggestIdx >= 0) {
                        e.preventDefault();
                        self._pickSuggest(input, items[self._suggestIdx].dataset.col);
                    } else if (e.key === 'Escape') {
                        box.classList.remove('active');
                    }
                });

                input.addEventListener('input', function () {
                    self._updateSuggest(input, box);
                });

                document.addEventListener('click', function (e) {
                    if (!e.target.closest('.pivot2-search-wrap')) box.classList.remove('active');
                });
            },

            _updateSuggest: function (input, box) {
                var val = input.value;
                var cur = input.selectionStart;
                var before = val.substring(0, cur);
                var lastOpen = before.lastIndexOf('{');
                var lastClose = before.lastIndexOf('}');
                if (lastOpen === -1 || lastClose > lastOpen) { box.classList.remove('active'); return; }

                var partial = before.substring(lastOpen + 1).toLowerCase();
                var filtered = this._colNames.filter(function (c) {
                    return c.name.toLowerCase().indexOf(partial) !== -1;
                });
                if (!filtered.length) { box.classList.remove('active'); return; }

                this._suggestIdx = 0;
                box.innerHTML = '';
                var self = this;
                filtered.forEach(function (c, i) {
                    var div = document.createElement('div');
                    div.className = 'pivot2-col-suggest-item' + (i === 0 ? ' selected' : '');
                    div.dataset.col = c.name;
                    div.innerHTML = c.name + '<span class="col-type">' + c.type + '</span>';
                    div.addEventListener('mousedown', function (e) {
                        e.preventDefault();
                        self._pickSuggest(input, c.name);
                    });
                    box.appendChild(div);
                });
                box.classList.add('active');
            },

            _hlSuggest: function (items) {
                var idx = this._suggestIdx;
                items.forEach(function (it, i) { it.classList.toggle('selected', i === idx); });
                if (items[idx]) items[idx].scrollIntoView({ block: 'nearest' });
            },

            _pickSuggest: function (input, colName) {
                var val = input.value;
                var cur = input.selectionStart;
                var before = val.substring(0, cur);
                var after = val.substring(cur);
                var lastOpen = before.lastIndexOf('{');
                var newBefore = before.substring(0, lastOpen) + '{' + colName + '} ';
                input.value = newBefore + after;
                input.selectionStart = input.selectionEnd = newBefore.length;
                var box = document.getElementById('pivot2-col-suggest');
                if (box) box.classList.remove('active');
                input.focus();
            },

            // ── Master→Slave Drilldown ──
            drilldown: function (e, row) {
                var w = document.getElementById('dg-pivot2-wrapper');
                if (!w) return;
                var cfgStr = w.dataset.drilldown;
                if (!cfgStr) return;

                var cfg;
                try { cfg = JSON.parse(cfgStr); } catch (e) { console.error('Drilldown: invalid config', e); return; }

                // Get record fields from clicked row
                var fieldsStr = row.dataset.recordFields;
                var fields = {};
                if (fieldsStr) {
                    try { fields = JSON.parse(fieldsStr); } catch (e) { /* ignore */ }
                }

                // Build URL params from dynamic base URL provided by host app
                var params = new URLSearchParams();

                var actionUrl = cfg.url || '';
                if (!actionUrl && cfg.target) {
                    // Fallback backward compatibility for .target
                    actionUrl = cfg.target;
                }

                if (actionUrl && !actionUrl.startsWith('/') && !actionUrl.startsWith('http')) {
                    // It is a relative path. Resolve it using BaseURL provided by the host.
                    var base = cfg.base_url || window.location.pathname + window.location.search;
                    var parts = base.split('?');
                    var basePath = parts[0];
                    var baseSearch = parts.length > 1 ? '?' + parts[1] : '';

                    // Host-specific logic abstraction: If the host uses query params for routing
                    // (e.g. ?name=folder/report), we append the relative target to the folder inside that param.
                    if (baseSearch) {
                        var baseParams = new URLSearchParams(baseSearch);
                        var routeParam = '';
                        var routeVal = '';

                        // Guess the routing parameter (usually the first one, or 'name')
                        if (baseParams.has('name')) { routeParam = 'name'; routeVal = baseParams.get('name'); }

                        if (routeParam) {
                            var slashIdx = routeVal.lastIndexOf('/');
                            var folder = slashIdx >= 0 ? routeVal.substring(0, slashIdx + 1) : '';
                            baseParams.set(routeParam, folder + actionUrl);
                            actionUrl = basePath + '?' + baseParams.toString();
                        } else {
                            // Standard relative URL resolution
                            var slashIdx = basePath.lastIndexOf('/');
                            var folder = slashIdx >= 0 ? basePath.substring(0, slashIdx + 1) : '/';
                            actionUrl = folder + actionUrl + baseSearch;
                        }
                    } else {
                        // Standard relative URL resolution
                        var slashIdx = basePath.lastIndexOf('/');
                        var folder = slashIdx >= 0 ? basePath.substring(0, slashIdx + 1) : '/';
                        actionUrl = folder + actionUrl;
                    }
                }

                if (actionUrl) {
                    var parts = actionUrl.split('?');
                    actionUrl = parts[0];
                    if (parts.length > 1) {
                        var existingParams = new URLSearchParams(parts[1]);
                        existingParams.forEach(function (v, k) { params.set(k, v); });
                    }
                }

                params.set('autorun', 'true');

                if (cfg.params) {
                    for (var paramName in cfg.params) {
                        var tmpl = cfg.params[paramName];
                        var val = tmpl;

                        // { { :master_param } } — resolve from master's form inputs
                        var masterMatch = tmpl.match(/^\{\{:(\w+)\}\}$/);
                        if (masterMatch) {
                            var inputName = masterMatch[1];
                            var el = document.querySelector('[name="' + inputName + '"]');
                            if (el) {
                                if (el.tomselect) {
                                    val = el.tomselect.getValue();
                                } else {
                                    val = el.value;
                                }
                            }
                        }

                        // { { column_name } } — resolve from clicked row's record fields
                        var colMatch = tmpl.match(/^\{\{(\w+)\}\}$/);
                        if (colMatch && !masterMatch) {
                            var colName = colMatch[1];
                            val = fields[colName] || '';
                        }

                        if (Array.isArray(val)) {
                            val.forEach(function (v) { params.append(paramName, v); });
                        } else {
                            params.set(paramName, val);
                        }
                    }
                }

                var url = actionUrl + '?' + params.toString();
                var winName = cfg.window || cfg.url_name || (cfg.target ? cfg.target.replace(/[^a-zA-Z0-9_]/g, '_') : null);

                // Allow user to force new tab/window via Ctrl+Click or Middle-Click
                if (e && (e.ctrlKey || e.metaKey || e.button === 1)) {
                    winName = '_blank';
                }

                if (!winName) {
                    console.error("Pivot2 Drilldown Error: Missing 'target' or 'window' in configuration.");
                    if (typeof showNotification === 'function') {
                        showNotification("Drilldown error: target configuration missing.", "error");
                    } else {
                        alert("Drilldown error: target configuration missing.");
                    }
                    return;
                }

                // Highlight clicked row
                w.querySelectorAll('tr.pivot2-row').forEach(function (r) {
                    r.classList.remove('pivot2-drilldown-active');
                });
                row.classList.add('pivot2-drilldown-active');

                var win = window.open(url, winName);
                if (win) {
                    win.focus();
                }
            }
        };
        window.Pivot2 = Pivot2;
        Pivot2.initAutocomplete();
    })();
