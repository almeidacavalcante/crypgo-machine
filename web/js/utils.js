// Utilitários gerais para o dashboard

/**
 * Formata valores monetários em BRL
 */
function formatCurrency(value) {
    if (value === null || value === undefined || value === '') {
        return 'R$ 0,00';
    }
    
    const num = parseFloat(value);
    if (isNaN(num)) {
        return 'R$ 0,00';
    }
    
    return new Intl.NumberFormat('pt-BR', {
        style: 'currency',
        currency: 'BRL'
    }).format(num);
}

/**
 * Formata timestamps para exibição
 */
function formatTimestamp(timestamp) {
    if (!timestamp) return '-';
    
    const date = new Date(timestamp);
    if (isNaN(date.getTime())) return '-';
    
    return new Intl.DateTimeFormat('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    }).format(date);
}

/**
 * Formata o último update
 */
function formatLastUpdate() {
    const now = new Date();
    return `Última atualização: ${now.toLocaleTimeString('pt-BR')}`;
}

/**
 * Trunca IDs longos para exibição
 */
function truncateId(id) {
    if (!id || id.length <= 8) return id;
    return `${id.substring(0, 8)}...`;
}

/**
 * Traduz status para português
 */
function translateStatus(status) {
    const translations = {
        'RUNNING': 'Rodando',
        'STOPPED': 'Parado',
        'PAUSED': 'Pausado'
    };
    return translations[status] || status;
}

/**
 * Traduz estratégias para português
 */
function translateStrategy(strategy) {
    const translations = {
        'MovingAverage': 'Média Móvel',
        'Breakout': 'Rompimento',
        'RSI': 'RSI',
        'MACD': 'MACD'
    };
    return translations[strategy] || strategy;
}

/**
 * Cria elemento de badge de status
 */
function createStatusBadge(status) {
    const span = document.createElement('span');
    span.className = `status-badge status-${status.toLowerCase()}`;
    span.textContent = translateStatus(status);
    return span;
}

/**
 * Cria elemento de badge de posição
 */
function createPositionBadge(positioned) {
    const span = document.createElement('span');
    span.className = `position-badge position-${positioned ? 'yes' : 'no'}`;
    span.textContent = positioned ? 'Sim' : 'Não';
    return span;
}

/**
 * Debounce para otimizar filtros
 */
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

/**
 * Show/hide loading state
 */
function setLoadingState(element, isLoading) {
    if (isLoading) {
        element.classList.add('metrics-loading');
    } else {
        element.classList.remove('metrics-loading');
    }
}

/**
 * Show notification (simple toast)
 */
function showNotification(message, type = 'info') {
    // Cria toast simples se não existir
    let toast = document.getElementById('toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 20px;
            border-radius: 6px;
            color: white;
            font-size: 0.9rem;
            z-index: 1000;
            opacity: 0;
            transform: translateX(100%);
            transition: all 0.3s ease;
        `;
        document.body.appendChild(toast);
    }
    
    // Define cor baseada no tipo
    const colors = {
        'info': '#007bff',
        'success': '#00ff88',
        'error': '#ff4444',
        'warning': '#ffc107'
    };
    
    toast.style.backgroundColor = colors[type] || colors.info;
    toast.textContent = message;
    
    // Mostra o toast
    toast.style.opacity = '1';
    toast.style.transform = 'translateX(0)';
    
    // Esconde após 3 segundos
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
    }, 3000);
}

/**
 * Calcula métricas agregadas dos bots
 */
function calculateMetrics(bots) {
    const metrics = {
        total: bots.length,
        active: 0,
        positioned: 0,
        totalCapital: 0,
        symbols: {}
    };
    
    bots.forEach(bot => {
        if (bot.status === 'RUNNING') {
            metrics.active++;
        }
        
        if (bot.positioned) {
            metrics.positioned++;
        }
        
        if (bot.initial_capital) {
            metrics.totalCapital += parseFloat(bot.initial_capital);
        }
        
        // Agrupa por símbolo
        if (bot.symbol) {
            if (!metrics.symbols[bot.symbol]) {
                metrics.symbols[bot.symbol] = 0;
            }
            metrics.symbols[bot.symbol]++;
        }
    });
    
    return metrics;
}

/**
 * Filtra bots baseado nos filtros ativos
 */
function filterBots(bots, filters) {
    return bots.filter(bot => {
        if (filters.symbol && bot.symbol !== filters.symbol) {
            return false;
        }
        
        if (filters.status && bot.status !== filters.status) {
            return false;
        }
        
        if (filters.strategy && bot.strategy !== filters.strategy) {
            return false;
        }
        
        return true;
    });
}

/**
 * Log de debug (apenas em desenvolvimento)
 */
function debugLog(...args) {
    if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        console.log('[Dashboard Debug]', ...args);
    }
}