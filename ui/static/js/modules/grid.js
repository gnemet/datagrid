/**
 * grid.js - Sorting, Resizing, and Table Features
 */
window.currentSort = window.currentSort || [];
window.draggingCol = window.draggingCol || null;

window.updateSortIcons = function () {
    $('.datagrid-table th').each(function () {
        const field = $(this).data('field');
        $(this).find('.sort-indicator').remove();
        $(this).attr('data-sort', 'NONE');

        const sortIdx = currentSort.findIndex(s => s.field === field);
        if (sortIdx !== -1) {
            const s = currentSort[sortIdx];
            const iconLib = $('meta[name="icon-library"]').attr('content') || 'FontAwesome';
            let iconClass;
            let libClass;

            if (iconLib === 'Phosphor') {
                libClass = 'ph';
                iconClass = s.dir === 'ASC' ? 'ph-caret-up' : 'ph-caret-down';
            } else {
                libClass = 'fas';
                iconClass = s.dir === 'ASC' ? 'fa-sort-up' : 'fa-sort-down';
            }

            let html = `<span class="sort-indicator"><i class="${libClass} ${iconClass}"></i>`;
            if (currentSort.length > 1) {
                html += `<span class="sort-rank-sub">${sortIdx + 1}</span>`;
            }
            html += `</span>`;
            $(this).append(html).addClass('sort-active').attr('data-sort', s.dir);
        } else {
            $(this).removeClass('sort-active').attr('data-sort', 'NONE');
        }
    });
};

window.applyRowStyles = function () {
    $('.datagrid-table tbody tr').each(function () {
        const style = $(this).attr('data-row-style');
        if (style) $(this).attr('style', style);
    });
};

window.initColumnChooser = function () {
    const $dropdown = $('#column-chooser-dropdown');
    if (!$dropdown.length) return;

    $dropdown.empty();
    $('.datagrid-table th').each(function () {
        const field = $(this).data('field');
        if (!field) return;

        const label = $(this).data('label') || $(this).text().trim();
        const isVisible = !$(this).hasClass('hidden-col');
        const colClass = getColClass($(this));

        const $item = $(`
            <div class="chooser-item">
                <label>
                    <input type="checkbox" data-col="${colClass}" ${isVisible ? 'checked' : ''}>
                    <span>${label}</span>
                </label>
            </div>
        `);
        $dropdown.append($item);
    });

    $dropdown.find('input').off('change').on('change', function () {
        const colClass = $(this).data('col');
        const checked = $(this).is(':checked');
        const escaped = escapeClass(colClass);
        if (checked) $(`.${escaped}`).removeClass('hidden-col');
        else $(`.${escaped}`).addClass('hidden-col');
        saveSettings();
    });
};

window.expandJSONKeys = function () {
    const table = $('.datagrid-table');
    const rows = table.find('tbody tr');
    const keys = new Set();

    rows.each(function () {
        let d = $(this).data('json');
        if (!d) return;
        if (typeof d === 'string') {
            try { d = JSON.parse(d); } catch (e) { return; }
        }

        const flatten = (obj, prefix = '') => {
            Object.keys(obj).forEach(k => {
                let val = obj[k];
                const key = prefix ? `${prefix}.${k}` : k;
                if (typeof val === 'string' && (val.startsWith('{') || val.startsWith('['))) {
                    try { val = JSON.parse(val); } catch (e) { }
                }
                if (val !== null && typeof val === 'object' && !Array.isArray(val)) flatten(val, key);
                else keys.add(key);
            });
        };
        flatten(d);
    });

    const visibleFields = new Set();
    table.find('thead th').each(function () {
        const f = $(this).data('field');
        if (f) visibleFields.add(f);
    });

    keys.forEach(key => {
        if (visibleFields.has(key)) return;
        const $th = $(`<th class="col-dyn-key" data-field="dyn-${key}" draggable="true">
            <span class="dg-dyn-label">${key}</span>
            <div class="resizer"></div>
        </th>`);
        table.find('thead tr').append($th);

        rows.each(function () {
            let data = $(this).data('json');
            if (typeof data === 'string') try { data = JSON.parse(data); } catch (e) { data = {}; }
            const getValue = (obj, path) => {
                const parts = path.split('.');
                let current = obj;
                for (const part of parts) {
                    if (current === undefined || current === null) return undefined;
                    if (typeof current === 'string' && (current.trim().startsWith('{') || current.trim().startsWith('['))) {
                        try { current = JSON.parse(current); } catch (e) { }
                    }
                    if (typeof current === 'object' && current !== null) current = current[part];
                    else return undefined;
                }
                return current;
            };
            const val = getValue(data, key);
            const display = (val === undefined || val === null) ? '-' : (typeof val === 'object' ? JSON.stringify(val) : val);
            $(this).append(`<td class="col-dyn-key col-number">${display}</td>`);
        });
    });
};

// --- Drag and Drop for Column Reordering ---
let draggedColClass = null;

$(document).on('dragstart', '.datagrid-table th', function (e) {
    if ($(e.target).hasClass('resizer')) return;
    draggedColClass = getColClass($(this));
    e.originalEvent.dataTransfer.setData('text/plain', draggedColClass);
    $(this).addClass('dg-dragging');
});

$(document).on('dragend', '.datagrid-table th', function () {
    $(this).removeClass('dg-dragging');
    $('.dg-drag-over').removeClass('dg-drag-over');
});

$(document).on('dragover', '.datagrid-table th', function (e) {
    e.preventDefault();
    const targetClass = getColClass($(this));
    if (targetClass !== draggedColClass) {
        $(this).addClass('dg-drag-over');
    }
});

$(document).on('dragleave', '.datagrid-table th', function () {
    $(this).removeClass('dg-drag-over');
});

$(document).on('drop', '.datagrid-table th', function (e) {
    e.preventDefault();
    const targetClass = getColClass($(this));
    if (!draggedColClass || targetClass === draggedColClass) return;

    const $table = $('.datagrid-table');
    const $thead = $table.find('thead tr');
    const $draggedTh = $thead.find(`.${escapeClass(draggedColClass)}`);
    const $targetTh = $thead.find(`.${escapeClass(targetClass)}`);

    // Determine if dropping before or after
    const targetIdx = $targetTh.index();
    const draggedIdx = $draggedTh.index();

    if (draggedIdx < targetIdx) {
        $draggedTh.insertAfter($targetTh);
    } else {
        $draggedTh.insertBefore($targetTh);
    }

    // Now move all TD cells
    $table.find('tbody tr').each(function () {
        const $tr = $(this);
        const $draggedTd = $tr.find(`.${escapeClass(draggedColClass)}`);
        const $targetTd = $tr.find(`.${escapeClass(targetClass)}`);
        if (draggedIdx < targetIdx) {
            $draggedTd.insertAfter($targetTd);
        } else {
            $draggedTd.insertBefore($targetTd);
        }
    });

    saveSettings();
    initColumnChooser();
});
