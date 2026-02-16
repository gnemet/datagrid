/**
 * Pivot Table Logic
 * Specifically for BI/Aggregation views
 */

(function () {
    console.log('Datagrid Pivot Logic Initialized');

    const PivotTable = {
        init: function () {
            this.bindEvents();
        },

        bindEvents: function () {
            // Future draggable logic for dimensions
        },

        formatValues: function () {
            // Apply specialized formatting for numbers in pivot
        }
    };

    // Auto-init on load
    document.addEventListener('DOMContentLoaded', () => {
        PivotTable.init();
    });

    // Handle HTMX swaps
    document.addEventListener('htmx:afterSwap', (evt) => {
        if (evt.detail.target.id === 'dg-pivot-wrapper') {
            PivotTable.init();
        }
    });

    window.PivotTable = PivotTable;
})();
