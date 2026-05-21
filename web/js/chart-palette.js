/**
 * QA Capsule chart color system — use for analytics widgets and Chart.js series.
 */
export const CHART_PALETTE = {
    brand: '#2563eb',
    series: [
        '#2563eb', // blue
        '#059669', // emerald
        '#d97706', // amber
        '#dc2626', // red
        '#7c3aed', // violet
        '#0891b2', // cyan
        '#ea580c', // orange
        '#4f46e5'  // indigo
    ],
    semantic: {
        info: '#2563eb',
        success: '#059669',
        warning: '#d97706',
        danger: '#dc2626',
        neutral: '#64748b'
    },
    doughnut: ['#dc2626', '#d97706', '#2563eb', '#059669'],
    metricPresets: [
        { label: 'Blue', value: '#2563eb' },
        { label: 'Emerald', value: '#059669' },
        { label: 'Amber', value: '#d97706' },
        { label: 'Red', value: '#dc2626' },
        { label: 'Violet', value: '#7c3aed' },
        { label: 'Cyan', value: '#0891b2' }
    ]
};

export function paletteColor(index) {
    return CHART_PALETTE.series[index % CHART_PALETTE.series.length];
}

export function defaultWidgetColors(type, index = 0) {
    if (type === 'doughnut') {
        return { color: CHART_PALETTE.doughnut[0], color2: CHART_PALETTE.doughnut[1] };
    }
    if (type === 'evolution') {
        return {
            color: CHART_PALETTE.series[0],
            color2: CHART_PALETTE.series[2],
            color3: CHART_PALETTE.series[1]
        };
    }
    return { color: paletteColor(index) };
}
