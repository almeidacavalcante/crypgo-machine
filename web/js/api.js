// Cliente API para comunicação com o backend

class ApiClient {
    constructor() {
        this.baseUrl = '/api/v1';
        this.headers = {
            'Content-Type': 'application/json'
        };
    }

    /**
     * Faz requisição HTTP genérica
     */
    async request(endpoint, options = {}) {
        const url = `${this.baseUrl}${endpoint}`;
        
        const config = {
            headers: this.headers,
            ...options
        };

        try {
            debugLog(`API Request: ${config.method || 'GET'} ${url}`);
            
            const response = await fetch(url, config);
            
            debugLog(`API Response: ${response.status} ${response.statusText}`);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            debugLog('API Data:', data);
            
            return {
                success: true,
                data: data,
                status: response.status
            };
            
        } catch (error) {
            console.error('API Error:', error);
            
            return {
                success: false,
                error: error.message,
                status: error.status || 0
            };
        }
    }

    /**
     * Lista todos os trading bots
     */
    async listBots() {
        return await this.request('/trading/list');
    }

    /**
     * Verifica saúde da API
     */
    async healthCheck() {
        try {
            const response = await fetch('/health', {
                method: 'GET',
                headers: this.headers
            });
            
            return {
                success: response.ok,
                status: response.status,
                data: response.ok ? await response.json() : null
            };
        } catch (error) {
            return {
                success: false,
                error: error.message,
                status: 0
            };
        }
    }

    /**
     * Faz ping para testar conectividade
     */
    async ping() {
        try {
            const start = Date.now();
            const response = await fetch('/health', {
                method: 'GET',
                headers: this.headers
            });
            const duration = Date.now() - start;
            
            return {
                success: response.ok,
                duration: duration,
                status: response.status
            };
        } catch (error) {
            return {
                success: false,
                error: error.message,
                duration: -1,
                status: 0
            };
        }
    }

    /**
     * Busca informações de um bot específico
     */
    async getBot(botId) {
        return await this.request(`/trading/bot/${botId}`);
    }

    /**
     * Busca logs de decisão de um bot
     */
    async getBotLogs(botId, limit = 10) {
        return await this.request(`/trading/bot/${botId}/logs?limit=${limit}`);
    }
}

// Instância global do cliente API
const apiClient = new ApiClient();

// Status de conexão
let connectionStatus = {
    connected: false,
    lastCheck: null,
    lastError: null
};

/**
 * Verifica status da conexão
 */
async function checkConnection() {
    const result = await apiClient.healthCheck();
    
    connectionStatus.connected = result.success;
    connectionStatus.lastCheck = new Date();
    connectionStatus.lastError = result.success ? null : result.error;
    
    // Atualiza indicador visual
    updateConnectionIndicator();
    
    return result;
}

/**
 * Atualiza o indicador visual de conexão
 */
function updateConnectionIndicator() {
    const statusDot = document.getElementById('connectionStatus');
    const lastUpdate = document.getElementById('lastUpdate');
    
    if (statusDot) {
        if (connectionStatus.connected) {
            statusDot.classList.add('connected');
            statusDot.title = 'Conectado';
        } else {
            statusDot.classList.remove('connected');
            statusDot.title = `Desconectado: ${connectionStatus.lastError || 'Erro desconhecido'}`;
        }
    }
    
    if (lastUpdate) {
        if (connectionStatus.connected) {
            lastUpdate.textContent = formatLastUpdate();
            lastUpdate.classList.remove('error');
        } else {
            lastUpdate.textContent = `Erro: ${connectionStatus.lastError || 'Sem conexão'}`;
            lastUpdate.classList.add('error');
        }
    }
}

/**
 * Wrapper para todas as chamadas de API que inclui verificação de conexão
 */
async function safeApiCall(apiCall, fallbackData = null) {
    try {
        const result = await apiCall();
        
        if (result.success) {
            connectionStatus.connected = true;
            connectionStatus.lastError = null;
            updateConnectionIndicator();
            return result;
        } else {
            connectionStatus.connected = false;
            connectionStatus.lastError = result.error;
            updateConnectionIndicator();
            
            if (fallbackData) {
                showNotification('Usando dados em cache devido a erro de conexão', 'warning');
                return { success: true, data: fallbackData };
            }
            
            throw new Error(result.error);
        }
    } catch (error) {
        connectionStatus.connected = false;
        connectionStatus.lastError = error.message;
        updateConnectionIndicator();
        
        if (fallbackData) {
            showNotification('Usando dados em cache devido a erro de conexão', 'warning');
            return { success: true, data: fallbackData };
        }
        
        throw error;
    }
}

// Adiciona estilo para indicador de erro
if (typeof document !== 'undefined') {
    const style = document.createElement('style');
    style.textContent = `
        #lastUpdate.error {
            color: #ff4444 !important;
        }
    `;
    document.head.appendChild(style);
}