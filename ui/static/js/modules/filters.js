/**
 * filters.js - Filtering and Search Logic
 */

window.dgToggleFilter = function (btn, field) {
    const $btn = $(btn);
    $btn.toggleClass('active');
    syncFilterForm(field, getValuesFromGroup(field));
};

window.dgToggleDropdown = function (btn) {
    const $dropdown = $(btn).closest('.dg-dropdown');
    const wasActive = $dropdown.hasClass('active');
    $('.dg-dropdown').removeClass('active');
    if (!wasActive) $dropdown.addClass('active');
};

window.dgCheckboxFilterUpdate = function (checkbox, field) {
    syncFilterForm(field, getValuesFromGroup(field));
};

function getValuesFromGroup(field) {
    // Check toggle group
    const $toggleGroup = $(`.dg-toggle-group[data-field="${field}"]`);
    if ($toggleGroup.length) {
        const $activeBtns = $toggleGroup.find('.dg-toggle-btn.active');
        const $allBtns = $toggleGroup.find('.dg-toggle-btn');
        if ($activeBtns.length === 0) return ["__NONE__"];
        if ($activeBtns.length === $allBtns.length) return [];
        return $activeBtns.map((i, el) => $(el).attr('data-value')).get();
    }


    // Check dropdown group
    const $dropdown = $(`.dg-dropdown[data-field="${field}"]`);
    if ($dropdown.length) {
        const $checked = $dropdown.find('input[type="checkbox"]:checked');
        const $all = $dropdown.find('input[type="checkbox"]');
        if ($checked.length === 0) return ["__NONE__"];
        if ($checked.length === $all.length) return [];
        return $checked.map((i, el) => $(el).val()).get();
    }


    return [];
}

function syncFilterForm(field, values) {
    const form = document.getElementById('datagrid-filter-form');
    if (!form) return;

    $(form).find(`input[name="${field}"]`).remove();

    values.forEach(val => {
        const input = document.createElement('input');
        input.type = 'hidden';
        input.name = field;
        input.value = val;
        input.className = 'dg-filter-hidden';
        form.appendChild(input);
    });

    $('#offset-input').val(0);
    htmx.trigger('#datagrid-filter-form', 'submit');
}

// Close dropdowns on outside click
$(document).on('click', function (e) {
    if (!$(e.target).closest('.dg-dropdown').length) {
        $('.dg-dropdown').removeClass('active');
    }
});

// Initialize all toggle buttons as active
$(document).on('htmx:load', function () {
    $('.dg-toggle-btn').addClass('active');
});

window.triggerPagination = function () {
    htmx.trigger('#datagrid-filter-form', 'submit');
};
