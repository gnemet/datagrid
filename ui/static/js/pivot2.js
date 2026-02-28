/**
 * Pivot2 — Hierarchical Collapse/Expand Logic
 * Tree-grid toggle for BI aggregation views.
 */
(function () {
    'use strict';

    const Pivot2 = {
        init: function () {
            // Start collapsed: only depth-0 rows visible (CSS handles initial .pivot2-hidden)
            this.bindEvents();
            this.initAutocomplete();
        },

        bindEvents: function () {
            // Re-bind after HTMX swaps
        },

        /**
         * Export visible rows to CSV
         */
        exportCSV: function () {
            var w = document.getElementById('dg-pivot2-wrapper');
            if (!w) return;
            var table = w.querySelector('.pivot2-table');
            if (!table) return;

            var csv = [];

            // Extract Headers
            var headers = [];
            table.querySelectorAll('thead th').forEach(function (th) {
                headers.push('"' + th.textContent.trim().replace(/"/g, '""') + '"');
            });
            csv.push(headers.join(';'));

            // Extract visible rows
            var rows = table.querySelectorAll('tbody tr.pivot2-row:not(.pivot2-hidden):not(.pivot2-filtered-out)');
            rows.forEach(function (row) {
                var rowData = [];
                row.querySelectorAll('td').forEach(function (td, idx) {
                    var text = td.textContent.trim();
                    if (idx === 0) {
                        text = (td.querySelector('.pivot2-text') || td).textContent.trim();
                        var depth = parseInt(row.dataset.depth, 10) || 0;
                        if (depth > 0) {
                            text = " ".repeat(depth * 4) + text; // pseudo-indent 
                        }
                    }
                    rowData.push('"' + text.replace(/"/g, '""') + '"');
                });
                csv.push(rowData.join(';'));
            });

            // Footer
            var footer = table.querySelector('tfoot tr.pivot2-grand-total-row');
            if (footer) {
                var footerData = [];
                footer.querySelectorAll('td').forEach(function (td) {
                    footerData.push('"' + td.textContent.trim().replace(/"/g, '""') + '"');
                });
                csv.push(footerData.join(';'));
            }

            var blob = new Blob(["\ufeff" + csv.join('\n')], { type: 'text/csv;charset=utf-8;' });
            var url = URL.createObjectURL(blob);
            var a = document.createElement('a');
            a.href = url;
            a.download = 'pivot_export_' + new Date().toISOString().slice(0, 10) + '.csv';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        },

        // Smart filter: supports {column} > value, {col} < val AND {col2} >= val2, or plain text
        localFilter: function (term) {
            var w = document.getElementById('dg-pivot2-wrapper');
            if (!w) return;
            var input = term.trim();
            var rows = w.querySelectorAll('tr.pivot2-row');
            if (!input) {
                // Clear filter: show depth-0 rows, hide deeper
                rows.forEach(function (r) {
                    r.classList.remove('pivot2-filtered-out');
                    if (parseInt(r.dataset.depth, 10) > 0) {
                        r.classList.add('pivot2-hidden');
                    } else {
                        r.classList.remove('pivot2-hidden');
                    }
                    var c = r.querySelector('.pivot2-chevron');
                    if (c) { c.classList.remove('pivot2-expanded'); c.classList.add('pivot2-collapsed'); }
                });
                return;
            }

            // Parse smart expressions: {col} op val [AND {col} op val ...]
            var exprRe = /\{([^}]+)\}\s*(>=|<=|!=|>|<|=|LIKE|IN|BETWEEN)\s*(.+?)(?=\s+AND\s+\{|$)/gi;
            var conditions = [];
            var m;
            // Reset regex explicitly just in case
            exprRe.lastIndex = 0;
            while ((m = exprRe.exec(input)) !== null) {
                conditions.push({ col: m[1].trim(), op: m[2], val: m[3].trim() });
            }

            if (conditions.length > 0) {
                this._smartFilter(w, rows, conditions);
            } else {
                this._textFilter(w, rows, input.toLowerCase());
            }
        },

        // Column-value expression filter
        _smartFilter: function (w, rows, conditions) {
            var headers = w.querySelectorAll('table thead th');
            var colMap = {};
            headers.forEach(function (th, idx) {
                colMap[th.textContent.trim().toLowerCase()] = idx;
            });

            var resolved = conditions.map(function (c) {
                var idx = colMap[c.col.toLowerCase()];
                return { idx: idx, op: c.op, val: c.val };
            }).filter(function (c) { return c.idx !== undefined; });

            if (resolved.length === 0) return;

            var rowArr = Array.from(rows);
            rowArr.forEach(function (r) { r.classList.add('pivot2-filtered-out'); r.classList.remove('pivot2-hidden'); });

            var ops = {
                '>': function (cVal, fVal) { return parseFloat(cVal) > parseFloat(fVal); },
                '<': function (cVal, fVal) { return parseFloat(cVal) < parseFloat(fVal); },
                '>=': function (cVal, fVal) { return parseFloat(cVal) >= parseFloat(fVal); },
                '<=': function (cVal, fVal) { return parseFloat(cVal) <= parseFloat(fVal); },
                '=': function (cVal, fVal) { return parseFloat(cVal) === parseFloat(fVal); },
                '!=': function (cVal, fVal) { return parseFloat(cVal) !== parseFloat(fVal); },
                'LIKE': function (cVal, fVal) {
                    var search = String(fVal).toLowerCase().replace(/%/g, '');
                    return String(cVal).toLowerCase().indexOf(search) !== -1;
                },
                'IN': function (cVal, fVal) {
                    var arr = String(fVal).replace(/^\(/, '').replace(/\)$/, '').split(',');
                    var valLower = String(cVal).toLowerCase();
                    return arr.some(function (item) { return item.trim().toLowerCase() === valLower; });
                },
                'BETWEEN': function (cVal, fVal) {
                    var parts = String(fVal).split(/ AND /i);
                    if (parts.length !== 2) return false;
                    var val = parseFloat(cVal);
                    return val >= parseFloat(parts[0]) && val <= parseFloat(parts[1]);
                }
            };

            rowArr.forEach(function (r, idx) {
                if (parseInt(r.dataset.depth, 10) !== 0) return;
                var cells = r.querySelectorAll('td');
                var match = resolved.every(function (cond) {
                    var cell = cells[cond.idx];
                    if (!cell) return false;

                    var cellText = cell.textContent.trim();
                    if (cond.idx === 0) {
                        var innerTextEl = cell.querySelector('.pivot2-text');
                        if (innerTextEl) cellText = innerTextEl.textContent.trim();
                    }

                    var opUpper = cond.op.toUpperCase();
                    var cVal = cellText;

                    if (['>', '<', '>=', '<=', '=', '!='].indexOf(opUpper) !== -1) {
                        cVal = cellText.replace(/[^\d.\-]/g, '');
                        if (cVal === '') return false; // avoid NaN matching defaults
                    }

                    return ops[opUpper] && ops[opUpper](cVal, cond.val);
                });

                if (match) {
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

        /**
         * Toggle children of a group row.
         * @param {HTMLElement} row - The clicked <tr> group row
         */
        toggle: function (row) {
            const key = row.dataset.key;
            const depth = parseInt(row.dataset.depth, 10);
            const chevron = row.querySelector('.pivot2-chevron');
            const isExpanding = chevron && chevron.classList.contains('pivot2-collapsed') ||
                (chevron && !chevron.classList.contains('pivot2-expanded'));

            // Find all direct children (depth = current + 1, key starts with this key)
            const table = row.closest('table');
            const rows = table.querySelectorAll('tr.pivot2-row');
            const rowArr = Array.from(rows);
            const rowIdx = rowArr.indexOf(row);

            if (chevron.classList.contains('pivot2-expanded')) {
                // Collapse: hide all descendants
                chevron.classList.remove('pivot2-expanded');
                chevron.classList.add('pivot2-collapsed');
                for (let i = rowIdx + 1; i < rowArr.length; i++) {
                    const child = rowArr[i];
                    const childDepth = parseInt(child.dataset.depth, 10);
                    if (childDepth <= depth) break; // Reached a sibling or parent
                    child.classList.add('pivot2-hidden');
                    // Also collapse any expanded sub-groups
                    const childChevron = child.querySelector('.pivot2-chevron');
                    if (childChevron) {
                        childChevron.classList.remove('pivot2-expanded');
                        childChevron.classList.add('pivot2-collapsed');
                    }
                }
            } else {
                // Expand: show direct children only (depth + 1)
                chevron.classList.remove('pivot2-collapsed');
                chevron.classList.add('pivot2-expanded');
                for (let i = rowIdx + 1; i < rowArr.length; i++) {
                    const child = rowArr[i];
                    const childDepth = parseInt(child.dataset.depth, 10);
                    if (childDepth <= depth) break; // Reached a sibling or parent
                    if (childDepth === depth + 1) {
                        child.classList.remove('pivot2-hidden');
                    }
                }
            }
        },

        /**
         * Expand all group rows.
         */
        expandAll: function () {
            const wrapper = document.getElementById('dg-pivot2-wrapper');
            if (!wrapper) return;
            wrapper.querySelectorAll('tr.pivot2-row').forEach(function (row) {
                row.classList.remove('pivot2-hidden');
                const chevron = row.querySelector('.pivot2-chevron');
                if (chevron) {
                    chevron.classList.remove('pivot2-collapsed');
                    chevron.classList.add('pivot2-expanded');
                }
            });
        },

        /**
         * Collapse all — show only depth-0 rows.
         */
        collapseAll: function () {
            const wrapper = document.getElementById('dg-pivot2-wrapper');
            if (!wrapper) return;
            wrapper.querySelectorAll('tr.pivot2-row').forEach(function (row) {
                const depth = parseInt(row.dataset.depth, 10);
                if (depth > 0) {
                    row.classList.add('pivot2-hidden');
                }
                const chevron = row.querySelector('.pivot2-chevron');
                if (chevron) {
                    chevron.classList.remove('pivot2-expanded');
                    chevron.classList.add('pivot2-collapsed');
                }
            });
        },

        // Column autocomplete for filter bar
        _colNames: null,
        _suggestIdx: -1,

        initAutocomplete: function () {
            var self = this;
            var input = document.querySelector('.pivot2-search');
            var box = document.getElementById('pivot2-col-suggest');
            if (!input || !box) return;

            // Collect column names from thead
            self._colNames = [];
            var w = document.getElementById('dg-pivot2-wrapper');
            if (w) {
                var ths = w.querySelectorAll('table thead th');
                // Also get first data row for type hints
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
                    self._highlightSuggest(items);
                } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    self._suggestIdx = Math.max(self._suggestIdx - 1, 0);
                    self._highlightSuggest(items);
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

            // Close on outside click
            document.addEventListener('click', function (e) {
                if (!e.target.closest('.pivot2-search-wrap')) {
                    box.classList.remove('active');
                }
            });
        },

        _updateSuggest: function (input, box) {
            var val = input.value;
            var cursorPos = input.selectionStart;
            var before = val.substring(0, cursorPos);

            // Check if we're inside an unclosed {
            var lastOpen = before.lastIndexOf('{');
            var lastClose = before.lastIndexOf('}');
            if (lastOpen === -1 || lastClose > lastOpen) {
                box.classList.remove('active');
                return;
            }

            var partial = before.substring(lastOpen + 1).toLowerCase();
            var filtered = this._colNames.filter(function (c) {
                return c.name.toLowerCase().indexOf(partial) !== -1;
            });

            if (filtered.length === 0) {
                box.classList.remove('active');
                return;
            }

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

        _highlightSuggest: function (items) {
            items.forEach(function (it, i) {
                it.classList.toggle('selected', i === this._suggestIdx);
            }.bind(this));
            if (items[this._suggestIdx]) {
                items[this._suggestIdx].scrollIntoView({ block: 'nearest' });
            }
        },

        _pickSuggest: function (input, colName) {
            var val = input.value;
            var cursorPos = input.selectionStart;
            var before = val.substring(0, cursorPos);
            var after = val.substring(cursorPos);

            var lastOpen = before.lastIndexOf('{');
            var newBefore = before.substring(0, lastOpen) + '{' + colName + '} ';
            input.value = newBefore + after;
            input.selectionStart = input.selectionEnd = newBefore.length;

            var box = document.getElementById('pivot2-col-suggest');
            if (box) box.classList.remove('active');
            input.focus();
        }
    };

    // Auto-init
    document.addEventListener('DOMContentLoaded', function () {
        Pivot2.init();
    });

    // Handle HTMX swaps
    document.addEventListener('htmx:afterSwap', function (evt) {
        if (evt.detail.target.id === 'bi-results' ||
            evt.detail.target.id === 'dg-pivot2-wrapper' ||
            evt.detail.target.querySelector('#dg-pivot2-wrapper')) {
            Pivot2.init();
        }
    });

    window.Pivot2 = Pivot2;
})();
