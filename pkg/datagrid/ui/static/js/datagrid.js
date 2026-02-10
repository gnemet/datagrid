/**
 * datagrid.js - Forensic DOM Table Handler
 * Advanced version with Multi-Sort, Column-Chooser, Drag-and-Drop, and Persistence
 */
var DATAGRID_SETTINGS_KEY = 'dg_settings_v1_';
var currentSort = currentSort || [];
var draggingCol = draggingCol || null;

function getSettingsKey() {
    const resource = window.datagridResource || 'default';
    return DATAGRID_SETTINGS_KEY + resource;
}

function saveSettings() {
    const resource = window.datagridResource;
    if (!resource) return;

    const settings = {
        sort: currentSort,
        limit: $('#limit-input').val(),
        columns: {},
        columnOrder: []
    };

    $('.datagrid-table th').each(function () {
        const field = $(this).data('field');
        const colClass = getColClass($(this));
        if (field) {
            settings.columnOrder.push(colClass);
            settings.columns[colClass] = {
                visible: !$(this).hasClass('hidden-col'),
                width: $(this)[0].style.width
            };
        }
    });

    localStorage.setItem(getSettingsKey(), JSON.stringify(settings));
}

function loadSettings() {
    const raw = localStorage.getItem(getSettingsKey());
    if (!raw) return;
    try {
        const s = JSON.parse(raw);
        if (s.sort) currentSort = Array.isArray(s.sort) ? s.sort : [s.sort];
        if (s.limit) {
            $('#limit-input').val(s.limit);
            $('#page-size-btn').text(s.limit);
        }
    } catch (e) { console.error("Load settings failed", e); }
}

function getColClass($el) {
    const classes = ($el.attr('class') || '').split(/\s+/);
    return classes.find(c => c.startsWith('col-')) || classes[0];
}

function escapeClass(cls) {
    if (!cls) return '';
    return cls.replace(/\./g, '\\.');
}

function applySettingsToTable() {
    const raw = localStorage.getItem(getSettingsKey());
    if (!raw) return;
    try {
        const s = JSON.parse(raw);
        const $table = $('.datagrid-table');
        if (!$table.length) return;
        const $thead = $table.find('thead tr');

        // 1. Reorder
        if (s.columnOrder && s.columnOrder.length > 0) {
            s.columnOrder.forEach(cls => {
                const $th = $thead.find(`.${escapeClass(cls)}`);
                if ($th.length) $thead.append($th);
            });

            $table.find('tbody tr').each(function () {
                const $tr = $(this);
                s.columnOrder.forEach(cls => {
                    const $td = $tr.find(`.${escapeClass(cls)}`);
                    if ($td.length) $tr.append($td);
                });
            });
        }

        // 2. Visibility & Width
        if (s.columns) {
            for (const [cls, cfg] of Object.entries(s.columns)) {
                const $els = $(`.${escapeClass(cls)}`);
                if (cfg.visible === false) $els.addClass('hidden-col');
                else $els.removeClass('hidden-col');
                if (cfg.width) $thead.find(`.${escapeClass(cls)}`).css('width', cfg.width);
            }
        }
    } catch (e) { console.error("Apply settings failed", e); }
}

function updateSortIcons() {
    $('.datagrid-table th').each(function () {
        const field = $(this).data('field');
        $(this).find('.sort-indicator').remove();
        $(this).attr('data-sort', 'NONE'); // Forensic Reset

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

            let html = `<span class="sort-indicator">
                            <i class="${libClass} ${iconClass}"></i>`;
            if (currentSort.length > 1) {
                html += `<span class="sort-rank-sub">${sortIdx + 1}</span>`;
            }
            html += `</span>`;
            $(this).append(html).addClass('sort-active').attr('data-sort', s.dir);
        } else {
            $(this).removeClass('sort-active').attr('data-sort', 'NONE');
        }
    });
}

function applyRowStyles() {
    $('.datagrid-table tbody tr').each(function () {
        const style = $(this).attr('data-row-style');
        if (style) {
            $(this).attr('style', style);
        }
    });
}

function initColumnChooser() {
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
        if (checked) {
            $(`.col-${escaped}, .${escaped}`).removeClass('hidden-col');
        } else {
            $(`.col-${escaped}, .${escaped}`).addClass('hidden-col');
        }
        saveSettings();
    });
}

function initDatagrid() {
    if (window.DATAGRID_INITIALIZED) {
        if ($('.datagrid-table').length) {
            applySettingsToTable();
            updateSortIcons();
            initColumnChooser();
        }
        return;
    }

    // 1. Column Chooser Toggle
    $(document).off('click.dg-chooser').on('click.dg-chooser', '#column-chooser-btn', function (e) {
        e.stopPropagation();
        $('#column-chooser-dropdown').toggleClass('hidden');
    });

    $(document).off('click.dg-close-chooser').on('click.dg-close-chooser', function () {
        $('#column-chooser-dropdown').addClass('hidden');
    });

    // 1.5 Sidebar Toggle
    $(document).off('click.dg-sidebar').on('click.dg-sidebar', '#toggle-sidebar-btn', function (e) {
        $('#datagrid-detail-sidebar').toggleClass('collapsed');
    });

    $('#column-chooser-dropdown').off('click').on('click', function (e) {
        e.stopPropagation();
    });

    // 2. Sorting & Hiding
    $(document).off('click.dg-sort').on('click.dg-sort', '.datagrid-table th.sortable', function (e) {
        if ($(e.target).hasClass('resizer')) return;

        // Shift+Click: Hide Column
        if (e.shiftKey) {
            const colClass = getColClass($(this));
            $(`.${escapeClass(colClass)}`).addClass('hidden-col');
            saveSettings();
            initColumnChooser(); // Update chooser UI
            return;
        }

        const field = $(this).data('field');
        const idx = currentSort.findIndex(s => s.field === field);
        let nextDir = 'ASC';
        if (idx !== -1) {
            if (currentSort[idx].dir === 'ASC') nextDir = 'DESC';
            else if (currentSort[idx].dir === 'DESC') nextDir = null; // Phase 3: None
        }

        if (e.ctrlKey) { // Multi-sort
            if (idx !== -1) {
                if (nextDir === null) currentSort.splice(idx, 1);
                else currentSort[idx].dir = nextDir;
            } else {
                currentSort.push({ field, dir: 'ASC' });
            }
        } else { // Single-sort
            if (idx !== -1 && currentSort.length === 1) {
                if (nextDir === null) currentSort = [];
                else currentSort[0].dir = nextDir;
            } else {
                currentSort = [{ field, dir: 'ASC' }];
            }
        }

        saveSettings();
        updateSortIcons();
        htmx.trigger('#datagrid-filter-form', 'submit');
    });

    // 3. Resizing
    $(document).off('mousedown.dg-resize').on('mousedown.dg-resize', '.resizer', function (e) {
        const th = $(this).closest('th');
        const startX = e.pageX;
        const startWidth = th.outerWidth();
        $(document).off('mousemove.datagrid-resize-move').on('mousemove.datagrid-resize-move', (me) => {
            th.css('width', (startWidth + (me.pageX - startX)) + 'px');
        });
        $(document).off('mouseup.datagrid-resize-up').on('mouseup.datagrid-resize-up', () => {
            $(document).off('mousemove.datagrid-resize-move mouseup.datagrid-resize-up');
            saveSettings();
        });
    });

    // 4. Dragging
    $(document).off('dragstart.dg-drag').on('dragstart.dg-drag', '.datagrid-table th', function (e) {
        draggingCol = this;
        $(this).addClass('dragging');
    });

    $(document).off('dragover.dg-drag').on('dragover.dg-drag', '.datagrid-table th', function (e) {
        e.preventDefault();
        $(this).addClass('drag-over');
    });

    $(document).off('dragleave.dg-drag').on('dragleave.dg-drag', '.datagrid-table th', function () {
        $(this).removeClass('drag-over');
    });

    $(document).off('drop.dg-drag').on('drop.dg-drag', '.datagrid-table th', function (e) {
        e.preventDefault();
        $(this).removeClass('drag-over');
        if (draggingCol && draggingCol !== this) {
            const srcIdx = $(draggingCol).index();
            const targetIdx = $(this).index();

            if (srcIdx < targetIdx) $(this).after(draggingCol);
            else $(this).before(draggingCol);

            const srcCls = escapeClass(getColClass($(draggingCol)));
            const targetCls = escapeClass(getColClass($(this)));

            $('.datagrid-table tbody tr').each(function () {
                const $srcTd = $(this).find('.' + srcCls);
                const $targetTd = $(this).find('.' + targetCls);
                if (srcIdx < targetIdx) $targetTd.after($srcTd);
                else $targetTd.before($srcTd);
            });
            saveSettings();
        }
    });

    $(document).off('dragend.dg-drag').on('dragend.dg-drag', '.datagrid-table th', function () {
        $(this).removeClass('dragging');
        $('.datagrid-table th').removeClass('drag-over');
        draggingCol = null;
    });

    // 5. Row Selection & Detail View
    $(document).off('click.dg-row').on('click.dg-row', '.datagrid-table tbody tr', function (e) {
        if ($(e.target).closest('button, a, input, [data-action]').length) return;

        $('.datagrid-table tbody tr').removeClass('selected');
        $(this).addClass('selected');

        const data = $(this).data('json');
        if (data) {
            updateLeftSidebar(data);
        }
    });

    $(document).off('click.dg-action').on('click.dg-action', '[data-action]', function (e) {
        const action = $(this).data('action');
        switch (action) {
            case 'close-sidebar':
                $('#right-sidebar').removeClass('active');
                break;
            case 'toggle-sidebar':
                $('#datagrid-detail-sidebar').toggleClass('collapsed');
                const isRCollapsed = $('#datagrid-detail-sidebar').hasClass('collapsed');
                // Update icon if present
                const iconLibSidebar = ($('meta[name="icon-library"]').attr('content') || 'FontAwesome').toLowerCase();
                const $rIcon = $(this).find('i');
                if ($rIcon.length) {
                    if (iconLibSidebar.includes('phosphor')) {
                        $rIcon.attr('class', isRCollapsed ? 'ph ph-caret-left' : 'ph ph-caret-right');
                    } else {
                        $rIcon.attr('class', isRCollapsed ? 'fas fa-angle-left' : 'fas fa-angle-right');
                    }
                }
                break;
            case 'switch-theme':
                const theme = $('html').attr('data-theme') === 'dark' ? 'light' : 'dark';
                $('html').attr('data-theme', theme);
                localStorage.setItem('dg_theme', theme);

                const iconLib = $('meta[name="icon-library"]').attr('content') || 'FontAwesome';
                const $icon = $(this).find('i');

                if (iconLib.toLowerCase().includes('phosphor')) {
                    $icon.attr('class', theme === 'dark' ? 'ph ph-sun' : 'ph ph-moon');
                } else {
                    $icon.attr('class', theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon');
                }
                break;
            case 'switch-lang':
                const nextLang = $(this).data('next-lang') || 'en';
                window.location.href = `/?lang=${nextLang}`;
                break;
            case 'expand-keys':
                const $btn = $(this);
                const isExpanded = $btn.hasClass('active');

                if (!isExpanded) {
                    expandJSONKeys();
                    $btn.addClass('active').attr('title', 'Collapse JSON Keys');
                    const iconLibExp = ($('meta[name="icon-library"]').attr('content') || 'FontAwesome').toLowerCase();
                    const $i = $btn.find('i');
                    if (iconLibExp.includes('phosphor')) {
                        $i.attr('class', 'ph ph-corners-in');
                    } else {
                        $i.attr('class', 'fas fa-compress-alt');
                    }
                } else {
                    $('.col-dyn-key').remove();
                    $btn.removeClass('active').attr('title', 'Expand JSON Keys');
                    const iconLibExp2 = ($('meta[name="icon-library"]').attr('content') || 'FontAwesome').toLowerCase();
                    const $i2 = $btn.find('i');
                    if (iconLibExp2.includes('phosphor')) {
                        $i2.attr('class', 'ph ph-corners-out');
                    } else {
                        $i2.attr('class', 'fas fa-expand-alt');
                    }
                }
                saveSettings();
                break;
        }
    });

    $(document).off('click', '#expand-keys-btn'); // Cleanup old listener if any

    // 6. Pagination
    $(document).off('click.dg-prev').on('click.dg-prev', '#prev-page-btn', function () {
        const offset = parseInt($('#offset-input').val()) || 0;
        const limit = parseInt($('#limit-input').val()) || 20;
        if (offset > 0) {
            $('#offset-input').val(Math.max(0, offset - limit));
            triggerPagination();
        }
    });

    $(document).off('click.dg-next').on('click.dg-next', '#next-page-btn', function () {
        const offset = parseInt($('#offset-input').val()) || 0;
        const limit = parseInt($('#limit-input').val()) || 20;
        const total = parseInt($('#pagination-metadata').data('total-count')) || 0;
        if (offset + limit < total) {
            $('#offset-input').val(offset + limit);
            triggerPagination();
        }
    });

    $(document).off('click.dg-psize').on('click.dg-psize', '#page-size-btn', function () {
        const sizes = [10, 20, 50, 100];
        const currentLimit = parseInt($('#limit-input').val()) || 20;
        const nextIdx = (sizes.indexOf(currentLimit) + 1) % sizes.length;
        const nextLimit = sizes[nextIdx];

        $('#limit-input').val(nextLimit);
        $('#offset-input').val(0);
        $(this).text(nextLimit);
        triggerPagination();
        saveSettings();
    });

    window.DATAGRID_INITIALIZED = true;
}

function triggerPagination() {
    htmx.trigger('#datagrid-filter-form', 'submit');
}

function expandJSONKeys() {
    const table = $('.datagrid-table');
    const rows = table.find('tbody tr');
    const keys = new Set();

    // 1. Collect all unique keys from Forensic DOM (data-json attribute)
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

                // Attempt to parse string as JSON
                if (typeof val === 'string' && (val.startsWith('{') || val.startsWith('['))) {
                    try { val = JSON.parse(val); } catch (e) { }
                }

                if (val !== null && typeof val === 'object' && !Array.isArray(val)) {
                    flatten(val, key);
                } else {
                    keys.add(key);
                }
            });
        };
        flatten(d);
    });

    // 2. Filter out keys that are already visible columns
    const visibleFields = new Set();
    table.find('thead th').each(function () {
        const f = $(this).data('field');
        if (f) visibleFields.add(f);
    });

    keys.forEach(key => {
        if (visibleFields.has(key)) return;

        // Add header
        const $th = $(`<th class="col-dyn-key" data-field="dyn-${key}" draggable="true">
            <span class="dg-dyn-label">${key}</span>
            <div class="resizer"></div>
        </th>`);
        table.find('thead tr').append($th);

        // Add cells
        rows.each(function () {
            let data = $(this).data('json');
            if (typeof data === 'string') {
                try { data = JSON.parse(data); } catch (e) { data = {}; }
            }

            const getValue = (obj, path) => {
                const parts = path.split('.');
                let current = obj;

                for (const part of parts) {
                    if (current === undefined || current === null) return undefined;

                    // If current is a string that looks like JSON, try to parse it
                    if (typeof current === 'string' && (current.trim().startsWith('{') || current.trim().startsWith('['))) {
                        try {
                            current = JSON.parse(current);
                        } catch (e) { }
                    }

                    // Proceed if it's an object
                    if (typeof current === 'object' && current !== null) {
                        current = current[part];
                    } else {
                        return undefined;
                    }
                }
                return current;
            };

            const val = getValue(data, key);
            const display = (val === undefined || val === null) ? '-' : (typeof val === 'object' ? JSON.stringify(val) : val);
            $(this).append(`<td class="col-dyn-key col-number">${display}</td>`);
        });
    });
}

function updateLeftSidebar(data) {
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
}

// Global initialization
if (document.readyState === 'loading') {
    $(document).ready(function () {
        loadSettings();
        initDatagrid();
    });
} else {
    loadSettings();
    initDatagrid();
}

// HTMX compatibility
document.body.addEventListener('htmx:configRequest', function (evt) {
    if (evt.detail.parameters && currentSort.length > 0) {
        evt.detail.parameters['sort'] = currentSort.map(s => `${s.field}:${s.dir}`).join(',');
    }
});

document.body.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target.id === 'datagrid-container' || evt.target.classList.contains('datagrid-table')) {
        const $meta = $('#pagination-metadata');
        if ($meta.length) {
            const limit = $meta.data('limit');
            const offset = $meta.data('offset');
            $('#limit-input').val(limit);
            $('#offset-input').val(offset);
            $('#page-size-btn').text(limit);

            const total = $meta.data('total-count');
            $('#prev-page-btn').prop('disabled', offset <= 0);
            $('#next-page-btn').prop('disabled', offset + limit >= total);
        }

        applySettingsToTable();
        applyRowStyles();
        updateSortIcons();
        initColumnChooser();

        // Remove initialization mask after layout settles
        setTimeout(() => {
            $('.datagrid-table').removeClass('dg-initializing');
        }, 50);
    }
});
