// vanarana - minimal interactive behaviors
document.addEventListener('DOMContentLoaded', function() {
    // Table row click to navigate
    document.querySelectorAll('.data-table tbody tr').forEach(function(row) {
        var link = row.querySelector('a');
        if (link) {
            row.style.cursor = 'pointer';
            row.addEventListener('click', function(e) {
                if (e.target.tagName !== 'A') {
                    window.location = link.href;
                }
            });
        }
    });
});
