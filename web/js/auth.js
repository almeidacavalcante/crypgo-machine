// Authentication module for JWT handling

class Auth {
    static TOKEN_KEY = 'crypgo_auth_token';
    static USER_KEY = 'crypgo_user_email';
    static API_BASE = '/api/v1/auth';

    /**
     * Login with email and password
     */
    static async login(email, password) {
        try {
            const response = await fetch(`${this.API_BASE}/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ email, password }),
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Erro de autenticação');
            }

            const data = await response.json();
            
            // Store token and user info
            localStorage.setItem(this.TOKEN_KEY, data.token);
            localStorage.setItem(this.USER_KEY, data.email);
            
            console.log('Login successful for:', data.email);
            return true;

        } catch (error) {
            console.error('Login error:', error);
            throw error;
        }
    }

    /**
     * Logout user
     */
    static logout() {
        localStorage.removeItem(this.TOKEN_KEY);
        localStorage.removeItem(this.USER_KEY);
        window.location.href = '/login.html';
    }

    /**
     * Get stored auth token
     */
    static getToken() {
        return localStorage.getItem(this.TOKEN_KEY);
    }

    /**
     * Get stored user email
     */
    static getUserEmail() {
        return localStorage.getItem(this.USER_KEY);
    }

    /**
     * Check if user is authenticated
     */
    static isAuthenticated() {
        return !!this.getToken();
    }

    /**
     * Validate current token
     */
    static async validateToken() {
        const token = this.getToken();
        if (!token) return false;

        try {
            const response = await fetch(`${this.API_BASE}/validate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ token }),
            });

            if (!response.ok) {
                this.logout();
                return false;
            }

            const data = await response.json();
            return data.valid;

        } catch (error) {
            console.error('Token validation error:', error);
            this.logout();
            return false;
        }
    }

    /**
     * Refresh auth token
     */
    static async refreshToken() {
        const token = this.getToken();
        if (!token) return false;

        try {
            const response = await fetch(`${this.API_BASE}/refresh`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ token }),
            });

            if (!response.ok) {
                this.logout();
                return false;
            }

            const data = await response.json();
            localStorage.setItem(this.TOKEN_KEY, data.token);
            return true;

        } catch (error) {
            console.error('Token refresh error:', error);
            this.logout();
            return false;
        }
    }

    /**
     * Make authenticated API request
     */
    static async apiRequest(url, options = {}) {
        const token = this.getToken();
        if (!token) {
            throw new Error('No authentication token');
        }

        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...options.headers,
        };

        const response = await fetch(url, {
            ...options,
            headers,
        });

        // Handle 401 unauthorized
        if (response.status === 401) {
            // Try to refresh token once
            const refreshed = await this.refreshToken();
            if (refreshed) {
                // Retry request with new token
                headers.Authorization = `Bearer ${this.getToken()}`;
                return fetch(url, { ...options, headers });
            } else {
                this.logout();
                throw new Error('Authentication expired');
            }
        }

        return response;
    }

    /**
     * Setup automatic token refresh
     */
    static setupAutoRefresh() {
        // Refresh token every 23 hours (tokens expire in 24h)
        setInterval(async () => {
            if (this.isAuthenticated()) {
                await this.refreshToken();
            }
        }, 23 * 60 * 60 * 1000); // 23 hours
    }

    /**
     * Redirect to login if not authenticated
     */
    static requireAuth() {
        if (!this.isAuthenticated()) {
            window.location.href = '/login.html';
            return false;
        }
        return true;
    }
}

// Setup auto-refresh when module loads
Auth.setupAutoRefresh();

// Export for use in other modules
window.Auth = Auth;