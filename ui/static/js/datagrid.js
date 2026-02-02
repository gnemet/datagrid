/**
 * datagrid.js - Forensic DOM Table Handler
 * Advanced version with Multi-Sort, Column-Chooser, Drag-and-Drop, and Persistence
 */
const DATAGRID_SETTINGS_KEY = 'dg_settings_v1_';
let currentSort = [];
let draggingCol = null;

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

    $('.meta-table th').each(function () {
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
    return cls.replace(/\./g, '\\.');
}

function applySettingsToTable() {
    const raw = localStorage.getItem(getSettingsKey());
    if (!raw) return;
    try {
        const s = JSON.parse(raw);
        const $table = $('.meta-table');
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
    $('.meta-table th').each(function () {
        const field = $(this).data('field');
        $(this).find('.sort-indicator').remove();

        const sortIdx = currentSort.findIndex(s => s.field === field);
        if (sortIdx !== -1) {
            const s = currentSort[sortIdx];
            const icon = s.dir === 'ASC' ? '▲' : '▼';
            let html = `<span class="sort-indicator">
                            <span class="sort-arrow">${icon}</span>`;
            if (currentSort.length > 1) {
                html += `<span class="sort-rank-sub">${sortIdx + 1}</span>`;
            }
            html += `</span>`;
            $(this).append(html).addClass('sort-active');
        } else {
            $(this).removeClass('sort-active');
        }
    });
}

function initColumnChooser() {
    const $dropdown = $('#column-chooser-dropdown');
    if (!$dropdown.length) return;

    $dropdown.empty();
    $('.meta-table th').each(function () {
        const field = $(this).data('field');
        if (!field) return;

        const label = $(this).text().trim();
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

    $dropdown.find('input').on('change', function () {
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

function initEvents() {
    // 1. Column Chooser Toggle
    $(document).on('click', '#column-chooser-btn', function (e) {
        e.stopPropagation();
        $('#column-chooser-dropdown').toggleClass('hidden');
    });

    $(document).on('click', function () {
        $('#column-chooser-dropdown').addClass('hidden');
    });

    // 1.5 Sidebar Toggle
    $(document).on('click', '#toggle-sidebar-btn', function (e) {
        $('#record-detail-sidebar').toggleClass('collapsed');
    });

    $('#column-chooser-dropdown').on('click', function (e) {
        e.stopPropagation();
    });

    // 2. Sorting & Hiding
    $(document).on('click', '.meta-table th.sortable', function (e) {
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
        htmx.trigger('#meta-list-container', 'load');
    });

    // 3. Resizing
    $(document).on('mousedown', '.resizer', function (e) {
        const th = $(this).closest('th');
        const startX = e.pageX;
        const startWidth = th.outerWidth();
        $(document).on('mousemove.datagrid-resize', (me) => {
            th.css('width', (startWidth + (me.pageX - startX)) + 'px');
        });
        $(document).on('mouseup.datagrid-resize', () => {
            $(document).off('.datagrid-resize');
            saveSettings();
        });
    });

    // 4. Dragging
    $(document).on('dragstart', '.meta-table th', function (e) {
        draggingCol = this;
        $(this).addClass('dragging');
    });

    $(document).on('dragover', '.meta-table th', function (e) {
        e.preventDefault();
        $(this).addClass('drag-over');
    });

    $(document).on('dragleave', '.meta-table th', function () {
        $(this).removeClass('drag-over');
    });

    $(document).on('drop', '.meta-table th', function (e) {
        e.preventDefault();
        $(this).removeClass('drag-over');
        if (draggingCol && draggingCol !== this) {
            const srcIdx = $(draggingCol).index();
            const targetIdx = $(this).index();

            if (srcIdx < targetIdx) $(this).after(draggingCol);
            else $(this).before(draggingCol);

            const srcCls = escapeClass(getColClass($(draggingCol)));
            const targetCls = escapeClass(getColClass($(this)));

            $('.meta-table tbody tr').each(function () {
                const $srcTd = $(this).find('.' + srcCls);
                const $targetTd = $(this).find('.' + targetCls);
                if (srcIdx < targetIdx) $targetTd.after($srcTd);
                else $targetTd.before($srcTd);
            });
            saveSettings();
        }
    });

    $(document).on('dragend', '.meta-table th', function () {
        $(this).removeClass('dragging');
        $('.meta-table th').removeClass('drag-over');
        draggingCol = null;
    });

    // 5. Row Selection & Detail View
    $(document).on('click', '.meta-table tbody tr', function (e) {
        if ($(e.target).closest('button, a, input, [data-action]').length) return;

        $('.meta-table tbody tr').removeClass('selected');
        $(this).addClass('selected');

        const data = $(this).data('json');
        if (data) {
            updateLeftSidebar(data);
        }
    });

    $(document).on('click', '[data-action]', function (e) {
        const action = $(this).data('action');
        switch (action) {
            case 'close-sidebar':
                $('#right-sidebar').removeClass('active');
                break;
            case 'toggle-sidebar':
                $('#record-detail-sidebar').toggleClass('collapsed');
                const isRCollapsed = $('#record-detail-sidebar').hasClass('collapsed');
                // Update icon if present
                const $rIcon = $(this).find('i.fa-angle-left, i.fa-angle-right');
                if ($rIcon.length) {
                    $rIcon.attr('class', isRCollapsed ? 'fas fa-angle-left' : 'fas fa-angle-right');
                }
                break;
            case 'switch-theme':
                const theme = $('html').attr('data-theme') === 'dark' ? 'light' : 'dark';
                $('html').attr('data-theme', theme);
                localStorage.setItem('dg_theme', theme);
                $(this).find('i').toggleClass('fa-moon fa-sun');
                break;
            case 'switch-lang':
                const langs = $(this).data('langs') || ['en'];
                const current = $(this).data('current') || 'en';
                const nextIdx = (langs.indexOf(current) + 1) % langs.length;
                const next = langs[nextIdx];
                $(this).data('current', next);
                window.location.href = `/?lang=${next}`; // Simplified for demo
                break;
        }
    });

    $(document).on('click', '#expand-keys-btn', function () {
        const $btn = $(this);
        const isExpanded = $btn.hasClass('active');

        if (!isExpanded) {
            expandJSONKeys();
            $btn.addClass('active').attr('title', 'Collapse JSON Keys').find('i').attr('class', 'fas fa-compress-alt');
        } else {
            $('.col-dyn-key').remove();
            $btn.removeClass('active').attr('title', 'Expand JSON Keys').find('i').attr('class', 'fas fa-expand-alt');
        }
        saveSettings();
    });
}

function expandJSONKeys() {
    const table = $('.meta-table');
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
        const $th = $(`<th class="col-dyn-key" data-field="dyn-${key}">
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
                return path.split('.').reduce((o, i) => {
                    if (!o) return undefined;
                    let val = o[i];
                    // Attempt to parse string as JSON mid-path
                    if (val && typeof val === 'string' && (val.startsWith('{') || val.startsWith('['))) {
                        try { val = JSON.parse(val); } catch (e) { }
                    }
                    return val;
                }, obj);
            };

            const val = getValue(data, key);
            const display = (val === undefined || val === null) ? '-' : (typeof val === 'object' ? JSON.stringify(val) : val);
            $(this).append(`<td class="col-dyn-key col-number">${display}</td>`);
        });
    });
}

// HTMX Config
document.body.addEventListener('htmx:configRequest', function (evt) {
    if (evt.detail.parameters) {
        evt.detail.parameters['sort'] = currentSort.map(s => `${s.field}:${s.dir}`).join(',');
    }
});

document.body.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target.id === 'meta-list-container' || evt.target.classList.contains('meta-table')) {
        applySettingsToTable();
        updateSortIcons();
        initColumnChooser();
    }
});

function updateLeftSidebar(data) {
    const container = $('#sidebar-detail-content');
    container.empty();

    let record = data;
    if (typeof data === 'string') {
        try { record = JSON.parse(data); } catch (e) { }
    }

    const $list = $('<div class="dg-details-list"></div>');

    const renderItems = (obj, prefix = '') => {
        Object.entries(obj).forEach(([key, val]) => {
            if (key.startsWith('_')) return;

            // Attempt to parse string as JSON
            let nested = val;
            if (typeof val === 'string' && (val.startsWith('{') || val.startsWith('['))) {
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

$(document).ready(function () {
    loadSettings();
    initEvents();
    // Initial apply if table already present
    if ($('.meta-table').length) {
        applySettingsToTable();
        updateSortIcons();
        initColumnChooser();
    }
});
