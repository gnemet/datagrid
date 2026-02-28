/**
 * Pivot2Chart — Native charting for Datagrid Pivot2.
 * Renders one chart per measure column, side by side.
 * Supports: horizontal bar & SVG donut pie, cycling on toggle.
 */
(function () {
    'use strict';

    var PALETTE = [
        '#89b4fa', '#a6e3a1', '#fab387', '#f38ba8', '#cba6f7',
        '#f9e2af', '#94e2d5', '#eba0ac', '#74c7ec', '#b4befe'
    ];

    var Pivot2Chart = {
        _mode: 'hidden', // 'hidden' | 'bar' | 'pie'

        toggle: function () {
            var container = document.getElementById('dg-pivot2-chart-container');
            if (!container) return;

            if (this._mode === 'hidden') {
                this._mode = 'bar';
                container.style.display = 'block';
                this._render(container);
            } else if (this._mode === 'bar') {
                this._mode = 'pie';
                this._render(container);
            } else {
                this._mode = 'hidden';
                container.style.display = 'none';
                container.innerHTML = '';
            }

            var btn = document.querySelector('[onclick*="Pivot2Chart.toggle"]');
            if (btn) {
                var icon = btn.querySelector('i');
                var label = btn.querySelector('span');
                if (this._mode === 'bar') {
                    if (icon) icon.className = 'fas fa-chart-bar';
                    if (label) label.textContent = 'Bar';
                } else if (this._mode === 'pie') {
                    if (icon) icon.className = 'fas fa-chart-pie';
                    if (label) label.textContent = 'Pie';
                } else {
                    if (icon) icon.className = 'fas fa-chart-bar';
                    if (label) label.textContent = 'Chart';
                }
            }
        },

        // Extract data for ALL measure columns (index 1+)
        _extractMultiData: function () {
            var w = document.getElementById('dg-pivot2-wrapper');
            if (!w) return [];

            // Get measure header names
            var ths = w.querySelectorAll('table thead th');
            var measures = [];
            ths.forEach(function (th, idx) {
                if (idx === 0) return; // skip label column
                measures.push({ name: th.textContent.trim(), colIdx: idx, labels: [], values: [] });
            });

            // Read depth-0 visible rows
            var rows = w.querySelectorAll('tbody tr.pivot2-row[data-depth="0"]:not(.pivot2-filtered-out)');
            rows.forEach(function (r) {
                var label = (r.querySelector('.pivot2-text') || {}).textContent || '';
                var cells = r.querySelectorAll('td');
                measures.forEach(function (m) {
                    if (cells[m.colIdx]) {
                        var raw = cells[m.colIdx].textContent.replace(/[^\d.\-]/g, '').trim();
                        var val = parseFloat(raw);
                        m.labels.push(label);
                        m.values.push(isNaN(val) ? 0 : val);
                    }
                });
            });

            // Filter out columns that are all zeros or non-numeric
            return measures.filter(function (m) {
                return m.values.some(function (v) { return v !== 0; });
            });
        },

        _render: function (container) {
            var datasets = this._extractMultiData();
            container.innerHTML = '';
            if (datasets.length === 0) {
                container.innerHTML = '<div style="padding:1rem; color:var(--dg-text-muted, #7f849c);">No data available for chart.</div>';
                return;
            }
            if (this._mode === 'bar') {
                this._drawMultiBar(container, datasets);
            } else if (this._mode === 'pie') {
                this._drawMultiPie(container, datasets);
            }
        },

        // ── Side-by-side Bar Charts ─────────────────────────────────────
        _drawMultiBar: function (container, datasets) {
            var html = '<div style="display: flex; gap: 1.5rem; flex-wrap: wrap; align-items: flex-start;">';

            datasets.forEach(function (ds, di) {
                var maxVal = Math.max.apply(null, ds.values);
                if (maxVal <= 0) maxVal = 1;
                var color = PALETTE[di % PALETTE.length];

                html += '<div style="flex: 1; min-width: 280px; max-width: 500px;">';
                html += '<div style="margin-bottom: 0.5rem; font-weight: 600; font-size: 0.85rem; color: ' + color + ';">' + ds.name + '</div>';
                html += '<div style="display: flex; flex-direction: column; gap: 0.35rem;">';

                for (var i = 0; i < ds.labels.length; i++) {
                    var pct = (Math.max(0, ds.values[i]) / maxVal) * 100;
                    var displayVal = ds.values[i].toLocaleString(undefined, { maximumFractionDigits: 2 });
                    var safeLabel = ds.labels[i].replace(/"/g, '&quot;');

                    html += '<div style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.78rem;">';
                    html += '  <div style="flex: 0 0 140px; text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--dg-text, #cdd6f4);" title="' + safeLabel + '">' + ds.labels[i] + '</div>';
                    html += '  <div style="flex: 1; display: flex; align-items: center;">';
                    html += '    <div style="width: ' + pct + '%; background: ' + color + '; height: 18px; border-radius: 3px; min-width: 2px; transition: width 0.3s ease; opacity: 0.85;"></div>';
                    html += '    <div style="margin-left: 0.4rem; color: var(--dg-text-muted, #a6adc8); font-size: 0.75rem;">' + displayVal + '</div>';
                    html += '  </div>';
                    html += '</div>';
                }

                html += '</div></div>';
            });

            html += '</div>';
            container.innerHTML = html;
        },

        // ── Side-by-side Pie Charts ─────────────────────────────────────
        _drawMultiPie: function (container, datasets) {
            var html = '<div style="display: flex; gap: 2rem; flex-wrap: wrap; align-items: flex-start;">';

            datasets.forEach(function (ds, di) {
                var total = 0;
                for (var i = 0; i < ds.values.length; i++) total += Math.max(0, ds.values[i]);
                if (total <= 0) return;

                var size = 180;
                var cx = size / 2, cy = size / 2;
                var outerR = 80, innerR = 40;
                var angle = -Math.PI / 2;
                var paths = '';

                for (var i = 0; i < ds.values.length; i++) {
                    var val = Math.max(0, ds.values[i]);
                    if (val === 0) continue;
                    var sliceAngle = (val / total) * 2 * Math.PI;
                    var endAngle = angle + sliceAngle;
                    var largeArc = sliceAngle > Math.PI ? 1 : 0;
                    var color = PALETTE[i % PALETTE.length];

                    var x1o = cx + outerR * Math.cos(angle);
                    var y1o = cy + outerR * Math.sin(angle);
                    var x2o = cx + outerR * Math.cos(endAngle);
                    var y2o = cy + outerR * Math.sin(endAngle);
                    var x1i = cx + innerR * Math.cos(endAngle);
                    var y1i = cy + innerR * Math.sin(endAngle);
                    var x2i = cx + innerR * Math.cos(angle);
                    var y2i = cy + innerR * Math.sin(angle);

                    var pct = ((val / total) * 100).toFixed(1);
                    var safeLabel = ds.labels[i].replace(/</g, '&lt;');

                    paths += '<path d="M ' + x1o + ' ' + y1o +
                        ' A ' + outerR + ' ' + outerR + ' 0 ' + largeArc + ' 1 ' + x2o + ' ' + y2o +
                        ' L ' + x1i + ' ' + y1i +
                        ' A ' + innerR + ' ' + innerR + ' 0 ' + largeArc + ' 0 ' + x2i + ' ' + y2i +
                        ' Z" fill="' + color + '" stroke="var(--dg-surface, #1e1e2e)" stroke-width="2" style="transition: opacity 0.2s;"' +
                        ' onmouseenter="this.style.opacity=0.8" onmouseleave="this.style.opacity=1">' +
                        '<title>' + safeLabel + ': ' + val.toLocaleString() + ' (' + pct + '%)</title></path>';

                    angle = endAngle;
                }

                // Compact legend
                var legend = '';
                for (var i = 0; i < ds.labels.length; i++) {
                    var val = Math.max(0, ds.values[i]);
                    if (val === 0) continue;
                    var pct = ((val / total) * 100).toFixed(1);
                    var color = PALETTE[i % PALETTE.length];
                    legend += '<div style="display: flex; align-items: center; gap: 0.3rem; font-size: 0.7rem; line-height: 1.6;">';
                    legend += '  <div style="width: 8px; height: 8px; border-radius: 2px; background: ' + color + '; flex-shrink: 0;"></div>';
                    legend += '  <div style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 140px; color: var(--dg-text, #cdd6f4);">' + ds.labels[i] + '</div>';
                    legend += '  <div style="color: var(--dg-text-muted, #a6adc8); margin-left: auto;">' + pct + '%</div>';
                    legend += '</div>';
                }

                var headerColor = PALETTE[di % PALETTE.length];
                html += '<div style="min-width: 200px;">';
                html += '<div style="margin-bottom: 0.3rem; font-weight: 600; font-size: 0.82rem; color: ' + headerColor + ';">' + ds.name + '</div>';
                html += '<div style="display: flex; align-items: flex-start; gap: 0.8rem;">';
                html += '  <svg width="' + size + '" height="' + size + '" viewBox="0 0 ' + size + ' ' + size + '">' + paths + '</svg>';
                html += '  <div style="max-height: ' + size + 'px; overflow-y: auto;">' + legend + '</div>';
                html += '</div></div>';
            });

            html += '</div>';
            container.innerHTML = html;
        }
    };

    window.Pivot2Chart = Pivot2Chart;
})();
