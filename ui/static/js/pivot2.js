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
        },

        bindEvents: function () {
            // Re-bind after HTMX swaps
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
        }
    };

    // Auto-init
    document.addEventListener('DOMContentLoaded', function () {
        Pivot2.init();
    });

    // Handle HTMX swaps
    document.addEventListener('htmx:afterSwap', function (evt) {
        if (evt.detail.target.id === 'dg-pivot2-wrapper' ||
            evt.detail.target.querySelector('#dg-pivot2-wrapper')) {
            Pivot2.init();
        }
    });

    window.Pivot2 = Pivot2;
})();
