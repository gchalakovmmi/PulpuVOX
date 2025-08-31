// Base JavaScript functionality
console.log('PulpuVOX base JavaScript loaded');

// Initialize Font Awesome if needed
if (typeof FontAwesome !== 'undefined') {
    FontAwesome.config.autoReplaceSvg = 'nest';
}

// Add any global event listeners or utilities here
document.addEventListener('DOMContentLoaded', function() {
    // Initialize tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
});
