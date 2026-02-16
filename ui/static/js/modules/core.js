/**
 * core.js - Initialization and HTMX Listeners
 */
// Defensive initialization mask removal
window.clearDatagridMask = function () {
    console.log("Clearing datagrid mask...");
    $('.dg-initializing').removeClass('dg-initializing');
    $('.datagrid-table').css('opacity', '1');
    window.DATAGRID_INITIALIZED = true;
};

window.initDatagrid = function () {
    if (window.DATAGRID_INITIALIZED) {

        if ($('.datagrid-table').length) {
            applySettingsToTable();
            updateSortIcons();
            initColumnChooser();
        }
        return;
    }

    // Toggle Handlers
    $(document).off('click.dg-chooser').on('click.dg-chooser', '#column-chooser-btn', function (e) {
        e.stopPropagation();
        $('#column-chooser-dropdown').toggleClass('hidden');
    });

    $(document).off('click.dg-close-chooser').on('click.dg-close-chooser', function () {
        $('#column-chooser-dropdown').addClass('hidden');
    });

    $(document).off('click.dg-sidebar').on('click.dg-sidebar', '#toggle-sidebar-btn', function (e) {
        $('#datagrid-detail-sidebar').toggleClass('collapsed');
    });

    $('#column-chooser-dropdown').on('click', e => e.stopPropagation());

    // Run mask removal
    setTimeout(window.clearDatagridMask, 300);
    setTimeout(window.clearDatagridMask, 1000);



    // Sorting Handler
    $(document).off('click.dg-sort').on('click.dg-sort', '.datagrid-table th.sortable', function (e) {
        if ($(e.target).hasClass('resizer')) return;
        if (e.shiftKey) {
            const colClass = getColClass($(this));
            $(`.${escapeClass(colClass)}`).addClass('hidden-col');
            saveSettings();
            initColumnChooser();
            return;
        }

        const field = $(this).data('field');
        const idx = currentSort.findIndex(s => s.field === field);
        let nextDir = 'ASC';
        if (idx !== -1) {
            if (currentSort[idx].dir === 'ASC') nextDir = 'DESC';
            else if (currentSort[idx].dir === 'DESC') nextDir = null;
        }

        if (e.ctrlKey) {
            if (idx !== -1) {
                if (nextDir === null) currentSort.splice(idx, 1);
                else currentSort[idx].dir = nextDir;
            } else currentSort.push({ field, dir: 'ASC' });
        } else {
            if (idx !== -1 && currentSort.length === 1) {
                if (nextDir === null) currentSort = [];
                else currentSort[0].dir = nextDir;
            } else currentSort = [{ field, dir: 'ASC' }];
        }
        saveSettings();
        updateSortIcons();
        htmx.trigger('#datagrid-filter-form', 'submit');
    });

    // Resizing Handler
    $(document).off('mousedown.dg-resize').on('mousedown.dg-resize', '.resizer', function (e) {
        const th = $(this).closest('th');
        const startX = e.pageX, startWidth = th.outerWidth();
        $(document).on('mousemove.datagrid-resize-move', me => th.css('width', (startWidth + (me.pageX - startX)) + 'px'));
        $(document).on('mouseup.datagrid-resize-up', () => {
            $(document).off('mousemove.datagrid-resize-move mouseup.datagrid-resize-up');
            saveSettings();
        });
    });

    // Row Selection
    $(document).off('click.dg-row').on('click.dg-row', '.datagrid-table tbody tr', function (e) {
        if ($(e.target).closest('button, a, input, [data-action]').length) return;
        $('.datagrid-table tbody tr').removeClass('selected');
        $(this).addClass('selected');
        const data = $(this).data('json');
        if (data) updateLeftSidebar(data);
    });

    // Action Handlers
    $(document).off('click.dg-action').on('click.dg-action', '[data-action]', function (e) {
        const action = $(this).data('action');
        switch (action) {
            case 'toggle-sidebar':
                $('#datagrid-detail-sidebar').toggleClass('collapsed');
                break;
            case 'switch-theme':
                const theme = $('html').attr('data-theme') === 'dark' ? 'light' : 'dark';
                $('html').attr('data-theme', theme);
                localStorage.setItem('dg_theme', theme);
                const iconLib = ($('meta[name="icon-library"]').attr('content') || 'FontAwesome').toLowerCase();
                const $icon = $(this).find('i');
                if (iconLib.includes('phosphor')) $icon.attr('class', theme === 'dark' ? 'ph ph-sun' : 'ph ph-moon');
                else $icon.attr('class', theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon');
                break;
        }
    });

    // Pagination Click Listeners
    $(document).on('click', '#prev-page-btn', () => {
        const offset = parseInt($('#offset-input').val()) || 0;
        const limit = parseInt($('#limit-input').val()) || 20;
        if (offset > 0) {
            $('#offset-input').val(Math.max(0, offset - limit));
            triggerPagination();
        }
    });

    $(document).on('click', '#next-page-btn', () => {
        const offset = parseInt($('#offset-input').val()) || 0;
        const limit = parseInt($('#limit-input').val()) || 20;
        const total = parseInt($('#pagination-metadata').data('total-count')) || 0;
        if (offset + limit < total) {
            $('#offset-input').val(offset + limit);
            triggerPagination();
        }
    });

    $(document).on('click', '#page-size-btn', function () {
        const sizes = [10, 20, 50, 100], currentLimit = parseInt($('#limit-input').val()) || 20;
        const nextLimit = sizes[(sizes.indexOf(currentLimit) + 1) % sizes.length];
        $('#limit-input').val(nextLimit);
        $('#offset-input').val(0);
        $(this).text(nextLimit);
        triggerPagination();
        saveSettings();
    });

    window.DATAGRID_INITIALIZED = true;
};

// Global initialization
$(document).ready(() => {
    loadSettings();
    initDatagrid();
});

// HTMX compatibility
document.body.addEventListener('htmx:configRequest', function (evt) {
    if (evt.detail.parameters && currentSort.length > 0) {
        evt.detail.parameters['sort'] = currentSort.map(s => `${s.field}:${s.dir}`).join(',');
    }
});

document.body.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target.id === 'datagrid-container' || evt.target.id === 'datagrid-main-view' ||
        evt.target.classList.contains('datagrid-table') || $(evt.target).find('.datagrid-table').length) {

        const $meta = $('#pagination-metadata');
        if ($meta.length) {
            const limit = $meta.data('limit'), offset = $meta.data('offset'), total = $meta.data('total-count');
            $('#limit-input').val(limit); $('#offset-input').val(offset); $('#page-size-btn').text(limit);
            $('#prev-page-btn').prop('disabled', offset <= 0);
            $('#next-page-btn').prop('disabled', offset + limit >= total);
        }
        applySettingsToTable();
        applyRowStyles();
        updateSortIcons();
        initColumnChooser();
        setTimeout(window.clearDatagridMask, 50);
    }
});

// HTMX load listener
document.addEventListener('htmx:load', function (evt) {
    if ($(evt.detail.elt).find('.datagrid-table').length || $(evt.detail.elt).hasClass('datagrid-table')) {
        setTimeout(window.clearDatagridMask, 50);
    }
});

// Final fallback for initializing mask
window.addEventListener('load', () => setTimeout(() => $('.datagrid-table').removeClass('dg-initializing'), 1000));
