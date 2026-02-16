/**
 * settings.js - Persistence and Global Utilities
 */
window.DATAGRID_SETTINGS_KEY = 'dg_settings_v1_';

window.getSettingsKey = function () {
    const resource = window.datagridResource || 'default';
    return DATAGRID_SETTINGS_KEY + resource;
};

window.saveSettings = function () {
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
};

window.loadSettings = function () {
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
};

window.getColClass = function ($el) {
    const classes = ($el.attr('class') || '').split(/\s+/);
    return classes.find(c => c.startsWith('col-')) || classes[0];
};

window.escapeClass = function (cls) {
    if (!cls) return '';
    return cls.replace(/\./g, '\\.');
};

window.applySettingsToTable = function () {
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
};
