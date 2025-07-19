# Dark Mode Design System for Crypgo Dashboard

## 1. Color Palette

### Primary Colors
```css
--primary-50: #e6f1ff;
--primary-100: #b3d4ff;
--primary-200: #80b8ff;
--primary-300: #4d9bff;
--primary-400: #1a7eff;
--primary-500: #0066ff; /* Main brand color */
--primary-600: #0052cc;
--primary-700: #003d99;
--primary-800: #002966;
--primary-900: #001433;
```

### Neutral Colors (Dark Mode Base)
```css
--neutral-50: #f8f9fa;
--neutral-100: #e9ecef;
--neutral-200: #dee2e6;
--neutral-300: #ced4da;
--neutral-400: #adb5bd;
--neutral-500: #6c757d;
--neutral-600: #495057;
--neutral-700: #343a40;
--neutral-800: #212529;
--neutral-900: #0a0c0e;

/* Dark mode specific */
--bg-primary: #0a0c0e;     /* Main background */
--bg-secondary: #12151a;   /* Card backgrounds */
--bg-tertiary: #1a1e24;    /* Elevated surfaces */
--bg-hover: #22272e;       /* Hover states */
```

### Accent Colors
```css
/* Success */
--success-light: #4ade80;
--success: #22c55e;
--success-dark: #16a34a;

/* Warning */
--warning-light: #fbbf24;
--warning: #f59e0b;
--warning-dark: #d97706;

/* Error */
--error-light: #f87171;
--error: #ef4444;
--error-dark: #dc2626;

/* Info */
--info-light: #60a5fa;
--info: #3b82f6;
--info-dark: #2563eb;
```

### Text Colors
```css
--text-primary: #f8f9fa;    /* High emphasis */
--text-secondary: #adb5bd;  /* Medium emphasis */
--text-tertiary: #6c757d;   /* Low emphasis */
--text-disabled: #495057;   /* Disabled state */
```

## 2. Typography

### Font Family
```css
--font-primary: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
--font-mono: 'JetBrains Mono', 'Monaco', 'Consolas', monospace;
```

### Font Sizes
```css
--text-xs: 0.75rem;     /* 12px */
--text-sm: 0.875rem;    /* 14px */
--text-base: 1rem;      /* 16px */
--text-lg: 1.125rem;    /* 18px */
--text-xl: 1.25rem;     /* 20px */
--text-2xl: 1.5rem;     /* 24px */
--text-3xl: 1.875rem;   /* 30px */
--text-4xl: 2.25rem;    /* 36px */
```

### Font Weights
```css
--font-light: 300;
--font-regular: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;
```

### Line Heights
```css
--leading-tight: 1.25;
--leading-normal: 1.5;
--leading-relaxed: 1.75;
```

## 3. Spacing System

```css
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
--space-16: 4rem;     /* 64px */
```

## 4. Border Radius

```css
--radius-sm: 4px;
--radius-md: 8px;
--radius-lg: 12px;
--radius-xl: 16px;
--radius-2xl: 24px;
--radius-full: 9999px;
```

## 5. Shadows & Elevation

### Dark Mode Shadows (Subtle glow effect)
```css
--shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.3), 0 0 0 1px rgba(255, 255, 255, 0.05);
--shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.05);
--shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 255, 255, 0.05);
--shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(255, 255, 255, 0.05);

/* Glow effects for interactive elements */
--glow-primary: 0 0 20px rgba(0, 102, 255, 0.4);
--glow-success: 0 0 20px rgba(34, 197, 94, 0.4);
--glow-error: 0 0 20px rgba(239, 68, 68, 0.4);
```

## 6. Component Styles

### Cards
```css
.card {
  background: var(--bg-secondary);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  transition: all 0.2s ease;
}

.card:hover {
  background: var(--bg-tertiary);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}
```

### Buttons
```css
.btn {
  font-weight: var(--font-medium);
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  transition: all 0.2s ease;
  border: none;
  cursor: pointer;
}

.btn-primary {
  background: var(--primary-500);
  color: white;
}

.btn-primary:hover {
  background: var(--primary-600);
  box-shadow: var(--glow-primary);
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.btn-ghost:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: rgba(255, 255, 255, 0.2);
}
```

### Inputs
```css
.input {
  background: var(--bg-primary);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.input:focus {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(0, 102, 255, 0.2);
}
```

### Tables
```css
.table {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.table th {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-weight: var(--font-medium);
  padding: var(--space-3) var(--space-4);
  text-align: left;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.table td {
  padding: var(--space-4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
}

.table tr:hover {
  background: var(--bg-hover);
}
```

## 7. Animation & Transitions

```css
/* Smooth transitions */
--transition-fast: 150ms ease;
--transition-base: 200ms ease;
--transition-slow: 300ms ease;

/* Hover animations */
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

/* Loading states */
@keyframes shimmer {
  0% { background-position: -200% center; }
  100% { background-position: 200% center; }
}

.skeleton {
  background: linear-gradient(
    90deg,
    var(--bg-secondary) 25%,
    var(--bg-tertiary) 50%,
    var(--bg-secondary) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
```

## 8. Best Practices

### Accessibility
- Maintain WCAG AA contrast ratios (4.5:1 for normal text, 3:1 for large text)
- Use semantic color naming that works in both light and dark modes
- Provide focus indicators for all interactive elements

### Performance
- Use CSS variables for easy theme switching
- Minimize paint operations with transform instead of position changes
- Use will-change sparingly for elements that will animate

### Visual Hierarchy
1. Use color and contrast to create clear hierarchy
2. Primary actions should use brand colors
3. Destructive actions should use error colors with confirmation
4. Disabled states should have reduced opacity (0.5)

### Dark Mode Specific Tips
1. Avoid pure black (#000) - use dark grays for better readability
2. Reduce contrast compared to light mode to prevent eye strain
3. Use subtle shadows and glows instead of heavy drop shadows
4. Add slight transparency to overlays and modals

## 9. Implementation Example

```css
:root[data-theme="dark"] {
  /* Apply all dark mode variables */
  --bg-primary: #0a0c0e;
  --bg-secondary: #12151a;
  --bg-tertiary: #1a1e24;
  /* ... rest of variables */
}

/* Component using the design system */
.dashboard-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  margin-bottom: var(--space-4);
  transition: all var(--transition-base);
}

.dashboard-card h2 {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-4);
}

.dashboard-card .metric {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--primary-400);
  font-variant-numeric: tabular-nums;
}
```

## 10. Visual References

### Inspiration Sources
- **Linear**: Clean gradients, subtle animations, glass morphism
- **Vercel**: Minimal dark theme with high contrast accents
- **Stripe**: Professional color system with excellent accessibility
- **GitHub**: Familiar dark mode patterns for developer tools
- **Tailwind UI**: Modern component patterns and spacing systems

### Key Visual Elements
1. **Glass morphism**: Subtle transparency with backdrop blur
2. **Gradient accents**: Smooth gradients for CTAs and highlights
3. **Micro-animations**: Subtle hover states and transitions
4. **Consistent spacing**: 8px grid system for alignment
5. **Soft shadows**: Glows and ambient lighting effects