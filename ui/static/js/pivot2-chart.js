/**
 * Pivot2Chart — Native CSS DOM bar charting for Datagrid Pivot2.
 */
(function () {
    'use strict';

    var Pivot2Chart = {
        // Toggle the chart visibility, draw if necessary
        toggle: function () {
            var container = document.getElementById('dg-pivot2-chart-container');
            if (!container) return;

            if (container.style.display === 'none' || !container.style.display) {
                container.style.display = 'block';
                this.draw(container);
            } else {
                container.style.display = 'none';
            }
        },

        // Draw a native HTML/CSS horizontal bar chart
        draw: function (container) {
            var w = document.getElementById('dg-pivot2-wrapper');
            if (!w) return;
            // Clear existing
            container.innerHTML = '';

            // Extract Data from depth-0 rows that are NOT filtered out
            var labels = [];
            var values = [];
            // Assuming we take the FIRST measure column (which is the 2nd td, index 1)
            var rows = w.querySelectorAll('tbody tr.pivot2-row[data-depth="0"]:not(.pivot2-filtered-out)');
            rows.forEach(function (r) {
                var label = (r.querySelector('.pivot2-text') || {}).textContent || '';
                var cells = r.querySelectorAll('td');
                if (cells.length > 1) {
                    var raw = cells[1].textContent.replace(/[^\d.\-]/g, '').trim();
                    var val = parseFloat(raw);
                    if (!isNaN(val)) {
                        labels.push(label);
                        values.push(val);
                    }
                }
            });

            if (labels.length === 0) {
                container.innerHTML = '<div style="padding:1rem; color:var(--dg-text-muted, #7f849c);">No data available for chart.</div>';
                return;
            }

            // Find max value for scaling
            var maxVal = Math.max.apply(null, values);
            if (maxVal <= 0) maxVal = 1; // Prevent division by zero if all values are 0 or negative

            // Build HTML
            var html = '<div style="margin-bottom: 1rem; font-weight: 600; font-size: 0.9rem; color: var(--dg-text, #cdd6f4);">Top Level Aggregation (First Measure)</div>';
            html += '<div style="display: flex; flex-direction: column; gap: 0.5rem;">';

            for (var i = 0; i < labels.length; i++) {
                var pct = (Math.max(0, values[i]) / maxVal) * 100;
                var displayVal = values[i].toLocaleString(undefined, { maximumFractionDigits: 2 });
                var safeLabel = labels[i].replace(/"/g, '&quot;');

                html += '<div style="display: flex; align-items: center; gap: 1rem; font-size: 0.82rem;">';
                // Label Wrapper
                html += '  <div style="flex: 0 0 220px; text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--dg-text, #cdd6f4);" title="' + safeLabel + '">' + labels[i] + '</div>';
                // Bar Wrapper
                html += '  <div style="flex: 1; display: flex; align-items: center;">';
                html += '    <div style="width: ' + pct + '%; background: var(--dg-accent, #89b4fa); height: 22px; border-radius: 4px; min-width: 2px; transition: width 0.3s ease;"></div>';
                html += '    <div style="margin-left: 0.6rem; color: var(--dg-text-muted, #a6adc8);">' + displayVal + '</div>';
                html += '  </div>';
                html += '</div>';
            }

            html += '</div>';
            container.innerHTML = html;
        }
    };

    window.Pivot2Chart = Pivot2Chart;
})();
