/**
 * QA Capsule chart color system — use for analytics widgets and Chart.js series.
 */
/** Chart series colors — aligned with Indigo Enterprise tokens in style.css */
export const CHART_PALETTE = {
    brand: '#4f46e5',
    series: [
        '#4f46e5',
        '#059669',
        '#d97706',
        '#dc2626',
        '#7c3aed',
        '#0891b2',
        '#ea580c',
        '#818cf8'
    ],
    semantic: {
        info: '#4f46e5',
        success: '#059669',
        warning: '#d97706',
        danger: '#dc2626',
        neutral: '#64748b'
    },
    doughnut: ['#dc2626', '#d97706', '#4f46e5', '#059669'],
    metricPresets: [
        { label: 'Indigo', value: '#4f46e5' },
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
