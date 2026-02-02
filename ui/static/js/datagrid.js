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

function applySettingsToTable() {
    const raw = localStorage.getItem(getSettingsKey());
    if (!raw) return;
    try {
        const s = JSON.parse(raw);
        const $table = $('.datagrid-table');
        const $thead = $table.find('thead tr');

        // 1. Reorder
        if (s.columnOrder && s.columnOrder.length > 0) {
            s.columnOrder.forEach(cls => {
                const $th = $thead.find(`.${cls}`);
                if ($th.length) $thead.append($th);
            });

            $table.find('tbody tr').each(function () {
                const $tr = $(this);
                s.columnOrder.forEach(cls => {
                    const $td = $tr.find(`.${cls}`);
                    if ($td.length) $tr.append($td);
                });
            });
        }

        // 2. Visibility & Width
        if (s.columns) {
            for (const [cls, cfg] of Object.entries(s.columns)) {
                const $els = $(`.${cls}`);
                if (cfg.visible === false) $els.addClass('hidden-col');
                else $els.removeClass('hidden-col');
                if (cfg.width) $thead.find(`.${cls}`).css('width', cfg.width);
            }
        }
    } catch (e) { console.error("Apply settings failed", e); }
}

function updateSortIcons() {
    $('.datagrid-table th').each(function () {
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
    $('.datagrid-table th').each(function() {
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

    $dropdown.find('input').on('change', function() {
        const colClass = $(this).data('col');
        const checked = $(this).is(':checked');
        if (checked) {
            $(`.${colClass}`).removeClass('hidden-col');
        } else {
            $(`.${colClass}`).addClass('hidden-col');
        }
        saveSettings();
    });
}

function initEvents() {
    // 1. Column Chooser Toggle
    $(document).on('click', '#column-chooser-btn', function(e) {
        e.stopPropagation();
        $('#column-chooser-dropdown').toggleClass('hidden');
    });

    $(document).on('click', function() {
        $('#column-chooser-dropdown').addClass('hidden');
    });

    $('#column-chooser-dropdown').on('click', function(e) {
        e.stopPropagation();
    });

    // 2. Sorting & Hiding
    $(document).on('click', '.datagrid-table th.sortable', function (e) {
        if ($(e.target).hasClass('resizer')) return;

        // Shift+Click: Hide Column
        if (e.shiftKey) {
            const colClass = getColClass($(this));
            $(`.${colClass}`).addClass('hidden-col');
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
        htmx.trigger('#filter-form', 'submit');
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
    $(document).on('dragstart', '.datagrid-table th', function (e) {
        draggingCol = this;
        $(this).addClass('dragging');
    });

    $(document).on('dragover', '.datagrid-table th', function (e) {
        e.preventDefault();
        $(this).addClass('drag-over');
    });

    $(document).on('dragleave', '.datagrid-table th', function () {
        $(this).removeClass('drag-over');
    });

    $(document).on('drop', '.datagrid-table th', function (e) {
        e.preventDefault();
        if (draggingCol && draggingCol !== this) {
            const srcIdx = $(draggingCol).index();
            const targetIdx = $(this).index();

            if (srcIdx < targetIdx) $(this).after(draggingCol);
            else $(this).before(draggingCol);

            const srcCls = getColClass($(draggingCol));
            const targetCls = getColClass($(this));

            $('.datagrid-table tbody tr').each(function () {
                const $srcTd = $(this).find('.' + srcCls);
                const $targetTd = $(this).find('.' + targetCls);
                if (srcIdx < targetIdx) $targetTd.after($srcTd);
                else $targetTd.before($srcTd);
            });
            saveSettings();
        }
    });

    $(document).on('dragend', '.datagrid-table th', function () {
        $('.datagrid-table th').removeClass('dragging drag-over');
    });
}

// HTMX Config
document.body.addEventListener('htmx:configRequest', function (evt) {
    if (evt.detail.parameters) {
        evt.detail.parameters['sort'] = currentSort.map(s => `${s.field}:${s.dir}`).join(',');
    }
});

document.body.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target.id === 'datagrid-container' || evt.target.classList.contains('datagrid-table')) {
        applySettingsToTable();
        updateSortIcons();
        initColumnChooser();
    }
});

$(document).ready(function () {
    loadSettings();
    initEvents();
    // Initial apply if table already present
    if ($('.datagrid-table').length) {
        applySettingsToTable();
        updateSortIcons();
        initColumnChooser();
    }
});
