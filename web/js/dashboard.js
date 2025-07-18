// Lógica principal do dashboard

class Dashboard {
    constructor() {
        this.bots = [];
        this.filteredBots = [];
        this.filters = {
            symbol: '',
            status: '',
            strategy: ''
        };
        this.logs = [];
        this.filteredLogs = [];
        this.logsFilters = {
            decision: '',
            symbol: ''
        };
        this.autoRefreshEnabled = true;
        this.autoRefreshInterval = null;
        this.refreshIntervalMs = 30000; // 30 segundos
        
        this.init();
    }

    /**
     * Inicializa o dashboard
     */
    async init() {
        debugLog('Inicializando dashboard...');
        
        // Verifica autenticação primeiro
        if (!Auth.requireAuth()) {
            return;
        }
        
        // Configura email do usuário no header
        const userEmail = Auth.getUserEmail();
        if (userEmail) {
            const userEmailElement = document.getElementById('userEmail');
            if (userEmailElement) {
                userEmailElement.textContent = userEmail;
            }
        }
        
        // Configura event listeners
        this.setupEventListeners();
        
        // Verifica conexão inicial
        await checkConnection();
        
        // Carrega dados iniciais
        await this.loadData();
        await this.loadLogs();
        
        // Inicia auto-refresh se habilitado
        this.setupAutoRefresh();
        
        debugLog('Dashboard inicializado com sucesso');
    }

    /**
     * Configura event listeners
     */
    setupEventListeners() {
        // Botão de refresh manual
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadData());
        }

        // Botão de logout
        const logoutBtn = document.getElementById('logoutBtn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', () => {
                if (confirm('Tem certeza que deseja sair?')) {
                    Auth.logout();
                }
            });
        }

        // Filtros
        const symbolFilter = document.getElementById('symbolFilter');
        const statusFilter = document.getElementById('statusFilter');
        const strategyFilter = document.getElementById('strategyFilter');

        if (symbolFilter) {
            symbolFilter.addEventListener('change', (e) => {
                this.filters.symbol = e.target.value;
                this.applyFilters();
            });
        }

        if (statusFilter) {
            statusFilter.addEventListener('change', (e) => {
                this.filters.status = e.target.value;
                this.applyFilters();
            });
        }

        if (strategyFilter) {
            strategyFilter.addEventListener('change', (e) => {
                this.filters.strategy = e.target.value;
                this.applyFilters();
            });
        }

        // Auto-refresh toggle
        const autoRefreshCheckbox = document.getElementById('autoRefresh');
        if (autoRefreshCheckbox) {
            autoRefreshCheckbox.addEventListener('change', (e) => {
                this.autoRefreshEnabled = e.target.checked;
                this.setupAutoRefresh();
            });
        }

        // Logs event listeners
        const refreshLogsBtn = document.getElementById('refreshLogsBtn');
        if (refreshLogsBtn) {
            refreshLogsBtn.addEventListener('click', () => this.loadLogs());
        }

        const logsDecisionFilter = document.getElementById('logsDecisionFilter');
        if (logsDecisionFilter) {
            logsDecisionFilter.addEventListener('change', (e) => {
                this.logsFilters.decision = e.target.value;
                this.applyLogsFilters();
            });
        }

        const logsSymbolFilter = document.getElementById('logsSymbolFilter');
        if (logsSymbolFilter) {
            logsSymbolFilter.addEventListener('change', (e) => {
                this.logsFilters.symbol = e.target.value;
                this.applyLogsFilters();
            });
        }
    }

    /**
     * Configura auto-refresh
     */
    setupAutoRefresh() {
        // Limpa intervalo existente
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
            this.autoRefreshInterval = null;
        }

        // Configura novo intervalo se habilitado
        if (this.autoRefreshEnabled) {
            this.autoRefreshInterval = setInterval(() => {
                this.loadData();
                this.loadLogs();
            }, this.refreshIntervalMs);
            
            debugLog(`Auto-refresh configurado para ${this.refreshIntervalMs / 1000}s`);
        } else {
            debugLog('Auto-refresh desabilitado');
        }
    }

    /**
     * Carrega dados da API
     */
    async loadData() {
        debugLog('Carregando dados...');
        
        try {
            // Mostra loading state
            this.setLoadingState(true);

            // Busca bots da API
            const result = await safeApiCall(() => apiClient.listBots(), this.bots);
            
            if (result.success) {
                this.bots = result.data || [];
                this.applyFilters();
                this.updateMetrics();
                
                showNotification('Dados atualizados com sucesso', 'success');
                debugLog(`${this.bots.length} bots carregados`);
            } else {
                throw new Error(result.error || 'Erro ao carregar dados');
            }

        } catch (error) {
            console.error('Erro ao carregar dados:', error);
            showNotification(`Erro ao carregar dados: ${error.message}`, 'error');
            
            // Se não temos dados, mostra estado vazio
            if (this.bots.length === 0) {
                this.showEmptyState();
            }
        } finally {
            this.setLoadingState(false);
        }
    }

    /**
     * Aplica filtros aos bots
     */
    applyFilters() {
        this.filteredBots = filterBots(this.bots, this.filters);
        this.updateTable();
        
        debugLog(`Filtros aplicados. ${this.filteredBots.length} de ${this.bots.length} bots exibidos`);
    }

    /**
     * Atualiza métricas no topo
     */
    updateMetrics() {
        const metrics = calculateMetrics(this.bots);
        
        // Atualiza cards de métricas
        const totalBots = document.getElementById('totalBots');
        const activeBots = document.getElementById('activeBots');
        const positionedBots = document.getElementById('positionedBots');
        const totalCapital = document.getElementById('totalCapital');

        if (totalBots) totalBots.textContent = metrics.total;
        if (activeBots) activeBots.textContent = metrics.active;
        if (positionedBots) positionedBots.textContent = metrics.positioned;
        if (totalCapital) totalCapital.textContent = formatCurrency(metrics.totalCapital);

        debugLog('Métricas atualizadas:', metrics);
    }

    /**
     * Atualiza tabela de bots
     */
    updateTable() {
        const tableBody = document.getElementById('botsTableBody');
        if (!tableBody) return;

        // Limpa tabela
        tableBody.innerHTML = '';

        if (this.filteredBots.length === 0) {
            // Mostra mensagem de vazio
            const row = document.createElement('tr');
            row.innerHTML = `
                <td colspan="10" class="empty-state">
                    <div class="empty-state-icon">🤖</div>
                    <div class="empty-state-message">Nenhum bot encontrado</div>
                    <div class="empty-state-description">
                        ${this.bots.length === 0 ? 'Nenhum trading bot configurado ainda.' : 'Tente ajustar os filtros acima.'}
                    </div>
                </td>
            `;
            tableBody.appendChild(row);
            return;
        }

        // Popula tabela com bots
        this.filteredBots.forEach(bot => {
            const row = this.createBotRow(bot);
            tableBody.appendChild(row);
        });
    }

    /**
     * Cria linha da tabela para um bot
     */
    createBotRow(bot) {
        const row = document.createElement('tr');
        row.className = 'fade-in';

        row.innerHTML = `
            <td>
                <span class="bot-id">${truncateId(bot.id)}</span>
            </td>
            <td>
                <span class="crypto-symbol">${bot.symbol || '-'}</span>
            </td>
            <td>
                ${bot.status ? createStatusBadge(bot.status).outerHTML : '-'}
            </td>
            <td>
                ${createPositionBadge(bot.positioned || false).outerHTML}
            </td>
            <td>
                ${translateStrategy(bot.strategy) || '-'}
            </td>
            <td>
                <span class="currency-value">${formatCurrency(bot.initial_capital)}</span>
            </td>
            <td>
                <span class="currency-value">${formatCurrency(bot.trade_amount)}</span>
            </td>
            <td>
                <span class="currency-value">
                    ${bot.entry_price ? formatCurrency(bot.entry_price) : '-'}
                </span>
            </td>
            <td>
                <span class="currency-small">${bot.minimum_profit_threshold || '-'}%</span>
            </td>
            <td>
                <span class="timestamp">${formatTimestamp(bot.created_at)}</span>
            </td>
        `;

        return row;
    }

    /**
     * Define estado de loading
     */
    setLoadingState(isLoading) {
        const refreshBtn = document.getElementById('refreshBtn');
        const metricsSection = document.querySelector('.metrics-section');
        
        if (refreshBtn) {
            refreshBtn.disabled = isLoading;
            refreshBtn.innerHTML = isLoading ? '⏳ Atualizando...' : '🔄 Atualizar';
        }
        
        if (metricsSection) {
            setLoadingState(metricsSection, isLoading);
        }
    }

    /**
     * Mostra estado vazio quando não há dados
     */
    showEmptyState() {
        const tableBody = document.getElementById('botsTableBody');
        if (!tableBody) return;

        tableBody.innerHTML = `
            <tr>
                <td colspan="10" class="empty-state">
                    <div class="empty-state-icon">📡</div>
                    <div class="empty-state-message">Não foi possível carregar os dados</div>
                    <div class="empty-state-description">
                        Verifique sua conexão e tente novamente.
                    </div>
                </td>
            </tr>
        `;

        // Zera métricas
        const elements = ['totalBots', 'activeBots', 'positionedBots', 'totalCapital'];
        elements.forEach(id => {
            const element = document.getElementById(id);
            if (element) element.textContent = '-';
        });
    }

    /**
     * Limpa e para o dashboard
     */
    destroy() {
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
            this.autoRefreshInterval = null;
        }
        debugLog('Dashboard destruído');
    }

    /**
     * Carrega logs de trading
     */
    async loadLogs() {
        debugLog('Carregando logs de trading...');
        
        try {
            const result = await safeApiCall(() => apiClient.listTradingLogs(this.logsFilters.decision, 50), []);
            
            if (result.success && result.data) {
                this.logs = result.data.logs || [];
                this.applyLogsFilters();
                debugLog(`${this.logs.length} logs carregados`);
            } else {
                throw new Error(result.error || 'Erro ao carregar logs');
            }

        } catch (error) {
            console.error('Erro ao carregar logs:', error);
            this.showEmptyLogsState();
        }
    }

    /**
     * Aplica filtros aos logs
     */
    applyLogsFilters() {
        this.filteredLogs = this.logs.filter(log => {
            const symbolMatch = !this.logsFilters.symbol || log.symbol === this.logsFilters.symbol;
            const decisionMatch = !this.logsFilters.decision || log.decision === this.logsFilters.decision;
            
            return symbolMatch && decisionMatch;
        });
        
        this.updateLogsTable();
        debugLog(`Filtros de logs aplicados. ${this.filteredLogs.length} de ${this.logs.length} logs exibidos`);
    }

    /**
     * Atualiza tabela de logs
     */
    updateLogsTable() {
        const tableBody = document.getElementById('logsTableBody');
        if (!tableBody) return;

        // Limpa tabela
        tableBody.innerHTML = '';

        if (this.filteredLogs.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td colspan="7" class="empty-state">
                    <div class="empty-state-icon">📊</div>
                    <div class="empty-state-message">Nenhum log encontrado</div>
                    <div class="empty-state-description">
                        ${this.logs.length === 0 ? 'Nenhum log de trading disponível ainda.' : 'Tente ajustar os filtros acima.'}
                    </div>
                </td>
            `;
            tableBody.appendChild(row);
            return;
        }

        // Preenche tabela com logs
        this.filteredLogs.forEach(log => {
            const row = document.createElement('tr');
            
            // Determina classe CSS baseada na decisão
            const decisionClass = {
                'BUY': 'decision-buy',
                'SELL': 'decision-sell',
                'HOLD': 'decision-hold'
            }[log.decision] || '';

            // Formata lucro/prejuízo
            let profitDisplay = '-';
            let profitClass = '';
            if (log.profit_percentage !== null && log.profit_percentage !== undefined) {
                const profit = log.profit_percentage;
                profitDisplay = `${profit >= 0 ? '+' : ''}${profit.toFixed(2)}%`;
                profitClass = profit >= 0 ? 'profit-positive' : 'profit-negative';
            }

            // Ícones para decisões
            const decisionIcons = {
                'BUY': '📈',
                'SELL': '📉', 
                'HOLD': '⏸️'
            };

            row.innerHTML = `
                <td>${formatDateTime(log.timestamp)}</td>
                <td class="symbol">${log.symbol || '-'}</td>
                <td class="decision ${decisionClass}">
                    ${decisionIcons[log.decision] || ''} ${log.decision}
                </td>
                <td class="price">${formatCurrency(log.current_price)}</td>
                <td class="price">${log.entry_price ? formatCurrency(log.entry_price) : '-'}</td>
                <td class="profit ${profitClass}">${profitDisplay}</td>
                <td class="strategy">${log.strategy_name}</td>
            `;

            tableBody.appendChild(row);
        });
    }

    /**
     * Mostra estado vazio para logs
     */
    showEmptyLogsState() {
        const tableBody = document.getElementById('logsTableBody');
        if (!tableBody) return;

        tableBody.innerHTML = `
            <tr>
                <td colspan="7" class="empty-state error">
                    <div class="empty-state-icon">⚠️</div>
                    <div class="empty-state-message">Erro ao carregar logs</div>
                    <div class="empty-state-description">Tente atualizar a página ou entre em contato com o suporte.</div>
                </td>
            </tr>
        `;
    }
}

// Inicializa dashboard quando o DOM estiver pronto
let dashboard = null;

document.addEventListener('DOMContentLoaded', () => {
    debugLog('DOM carregado, inicializando dashboard...');
    dashboard = new Dashboard();
});

// Cleanup ao sair da página
window.addEventListener('beforeunload', () => {
    if (dashboard) {
        dashboard.destroy();
    }
});

// Trata erros globais
window.addEventListener('error', (event) => {
    console.error('Erro global:', event.error);
    showNotification('Erro inesperado no dashboard', 'error');
});

// Trata erros de Promise rejeitadas
window.addEventListener('unhandledrejection', (event) => {
    console.error('Promise rejeitada:', event.reason);
    showNotification('Erro de conexão', 'error');
});