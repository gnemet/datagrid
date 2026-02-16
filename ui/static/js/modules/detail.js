/**
 * detail.js - Sidebar and Detail Panel Logic
 */
window.updateLeftSidebar = function (data) {
    const container = $('#datagrid-detail-content');
    if (!container.length) return;
    container.empty();

    let record = data;
    if (typeof data === 'string') {
        try { record = JSON.parse(data); } catch (e) { }
    }

    const $list = $('<div class="dg-details-list"></div>');

    const renderItems = (obj, prefix = '') => {
        if (!obj || typeof obj !== 'object') return;
        Object.entries(obj).forEach(([key, val]) => {
            if (key.startsWith('_')) return;

            let nested = val;
            if (typeof val === 'string' && (val.trim().startsWith('{') || val.trim().startsWith('['))) {
                try { nested = JSON.parse(val); } catch (e) { }
            }

            if (nested !== null && typeof nested === 'object' && !Array.isArray(nested)) {
                renderItems(nested, prefix ? `${prefix}.${key}` : key);
            } else {
                const label = prefix ? `${prefix}.${key}` : key;
                const display = (val === null || val === undefined) ? '-' : (typeof val === 'object' ? JSON.stringify(val) : val);
                const $item = $(`
                    <div class="dg-detail-item">
                        <div class="dg-detail-label">${label}</div>
                        <div class="dg-detail-value">${display}</div>
                    </div>
                `);
                $list.append($item);
            }
        });
    };

    renderItems(record);
    container.append($list);
};
