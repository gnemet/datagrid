const SETTINGS_KEY_PREFIX = 'datagrid_settings_';
let currentSort = [];
let isResizing = false;

function getSettingsKey() {
    return SETTINGS_KEY_PREFIX + (window.datagridResource || 'default');
}

function saveSettings() {
    const settings = {
        sort: currentSort,
        limit: $('#limit-input').val(),
        columns: {},
        columnOrder: []
    };

    $('.meta-table th').each(function() {
        const field = $(this).data('field');
        if (field) {
            const classes = $(this).attr('class') || '';
            const colClass = classes.split(/\s+/).find(c => c.startsWith('col-'));
            settings.columnOrder.push(colClass || field);
            settings.columns[colClass || field] = {
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
        if (s.sort) currentSort = s.sort;
        if (s.limit) $('#limit-input').val(s.limit);
        // Ordering and visibility applied after HTMX swap
    } catch(e) { console.error("Load settings failed", e); }
}

function updateSortIcons() {
    $('.meta-table th').each(function() {
        const field = $(this).data('field');
        $(this).find('.sort-icon, .sort-index').remove();
        
        const sortIdx = currentSort.findIndex(s => s.field === field);
        if (sortIdx !== -1) {
            const s = currentSort[sortIdx];
            const icon = s.dir === 'ASC' ? '▲' : '▼'; // Minimalist style
            let html = `<span class="sort-icon ml-1">${icon}</span>`;
            if (currentSort.length > 1) {
                html += `<span class="sort-index">${sortIdx + 1}</span>`;
            }
            $(this).append(html).addClass('sort-active');
        } else {
            $(this).removeClass('sort-active');
        }
    });
}

// HTMX Trigger with Multiple Sort Support
document.body.addEventListener('htmx:configRequest', function(evt) {
    if (evt.detail.parameters) {
        evt.detail.parameters['sort'] = currentSort.map(s => `${s.field}:${s.dir}`);
    }
});

$(document).on('click', '.meta-table th', function(e) {
    if ($(e.target).hasClass('resizer')) return;
    const field = $(this).data('field');
    if (!field || $(this).data('sortable') === false) return;

    if (e.ctrlKey) {
        const idx = currentSort.findIndex(s => s.field === field);
        if (idx !== -1) {
            if (currentSort[idx].dir === 'ASC') currentSort[idx].dir = 'DESC';
            else currentSort.splice(idx, 1); // 3-phase: ASC -> DESC -> NONE
        } else {
            currentSort.push({ field: field, dir: 'ASC' });
        }
    } else {
        if (currentSort.length === 1 && currentSort[0].field === field) {
            if (currentSort[0].dir === 'ASC') currentSort[0].dir = 'DESC';
            else currentSort = []; 
        } else {
            currentSort = [{ field: field, dir: 'ASC' }];
        }
    }

    saveSettings();
    updateSortIcons();
    htmx.trigger('#filter-form', 'submit');
});

// Row Selection & Sidebar
$(document).on('click', '.meta-table tbody tr', function() {
    $('.meta-table tr').removeClass('selected');
    $(this).addClass('selected');
    showDetails($(this).data('json'));
});

function showDetails(json) {
    if (!json) return;
    const data = typeof json === 'string' ? JSON.parse(json) : json;
    let html = '<div class="sidebar-details-grid">';
    for (const [k, v] of Object.entries(data)) {
        if (k.startsWith('_')) continue;
        html += `<div class="detail-row"><span class="detail-label">${k}</span><span class="detail-value">${v}</span></div>`;
    }
    html += '</div>';
    $('#sidebar-details').html(html);
    $('#right-sidebar').addClass('active');
}

// Resizing logic
$(document).on('mousedown', '.resizer', function(e) {
    isResizing = true;
    const th = $(this).closest('th');
    const startX = e.pageX;
    const startWidth = th.outerWidth();
    
    $(document).on('mousemove.resizer', function(e) {
        const w = startWidth + (e.pageX - startX);
        th.css('width', w + 'px');
    });
    
    $(document).on('mouseup.resizer', function() {
        isResizing = false;
        $(document).off('.resizer');
        saveSettings();
    });
});

$(document).ready(function() {
    loadSettings();
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        if (evt.target.id === 'meta-list-container') {
            updateSortIcons();
        }
    });
});
