// CasDash Frontend Application
class CasDash {
    constructor() {
        this.websocket = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.init();
    }

    init() {
        this.setupWebSocket();
        this.setupEventListeners();
        this.setupServiceCards();
        this.setupAutoRefresh();
        this.setupNotifications();
    }

    // WebSocket Connection
    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/status`;

        this.websocket = new WebSocket(wsUrl);

        this.websocket.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.showNotification('Connected to real-time updates', 'success');
        };

        this.websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket disconnected');
            this.reconnectWebSocket();
        };

        this.websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    reconnectWebSocket() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.pow(2, this.reconnectAttempts) * 1000; // Exponential backoff

            setTimeout(() => {
                console.log(`Reconnecting WebSocket (attempt ${this.reconnectAttempts})`);
                this.setupWebSocket();
            }, delay);
        } else {
            this.showNotification('Lost connection to server. Please refresh the page.', 'error');
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'service_status':
                this.updateServiceStatus(data.payload);
                break;
            case 'service_update':
                this.updateServiceCard(data.payload);
                break;
            case 'notification':
                this.showNotification(data.payload.message, data.payload.type);
                break;
            case 'monitoring_data':
                this.updateMonitoringData(data.payload);
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }

    // Event Listeners
    setupEventListeners() {
        // Mobile menu toggle
        const menuToggle = document.querySelector('.menu-toggle');
        const sidebar = document.querySelector('.sidebar');

        if (menuToggle && sidebar) {
            menuToggle.addEventListener('click', () => {
                sidebar.classList.toggle('open');
            });
        }

        // Close sidebar when clicking outside on mobile
        document.addEventListener('click', (e) => {
            if (sidebar && !sidebar.contains(e.target) && !menuToggle.contains(e.target)) {
                sidebar.classList.remove('open');
            }
        });

        // Form submissions
        document.addEventListener('submit', (e) => {
            if (e.target.classList.contains('ajax-form')) {
                e.preventDefault();
                this.handleAjaxForm(e.target);
            }
        });

        // Service card clicks
        document.addEventListener('click', (e) => {
            const serviceCard = e.target.closest('.service-card');
            if (serviceCard && !e.target.closest('.btn')) {
                const serviceId = serviceCard.dataset.serviceId;
                if (serviceId) {
                    window.location.href = `/services/${serviceId}`;
                }
            }
        });

        // Theme toggle
        const themeToggle = document.querySelector('.theme-toggle');
        if (themeToggle) {
            themeToggle.addEventListener('click', () => {
                this.toggleTheme();
            });
        }

        // Search functionality
        const searchInput = document.querySelector('.search-input');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.filterServices(e.target.value);
            });
        }
    }

    // Service Cards
    setupServiceCards() {
        const cards = document.querySelectorAll('.service-card');
        cards.forEach(card => {
            // Add hover effects
            card.addEventListener('mouseenter', () => {
                card.style.transform = 'translateY(-4px)';
            });

            card.addEventListener('mouseleave', () => {
                card.style.transform = 'translateY(0)';
            });
        });
    }

    updateServiceStatus(statusData) {
        const card = document.querySelector(`[data-service-id="${statusData.id}"]`);
        if (!card) return;

        const statusIndicator = card.querySelector('.service-status');
        const responseTimeEl = card.querySelector('.response-time');

        // Update status indicator
        if (statusIndicator) {
            statusIndicator.className = 'service-status';
            statusIndicator.classList.add(`status-${statusData.status}`);
        }

        // Update card border color
        card.className = card.className.replace(/\b(online|offline|warning|maintenance)\b/g, '');
        card.classList.add(statusData.status);

        // Update response time
        if (responseTimeEl && statusData.responseTime) {
            responseTimeEl.textContent = `${statusData.responseTime}ms`;
        }

        // Update last checked time
        const lastCheckedEl = card.querySelector('.last-checked');
        if (lastCheckedEl) {
            lastCheckedEl.textContent = this.formatTimeAgo(new Date());
        }
    }

    updateServiceCard(serviceData) {
        const card = document.querySelector(`[data-service-id="${serviceData.id}"]`);
        if (!card) return;

        // Update service name
        const nameEl = card.querySelector('.service-name');
        if (nameEl) {
            nameEl.textContent = serviceData.name;
        }

        // Update service URL
        const urlEl = card.querySelector('.service-url');
        if (urlEl) {
            urlEl.textContent = serviceData.url;
            urlEl.href = serviceData.url;
        }

        // Update service icon
        const iconEl = card.querySelector('.service-icon');
        if (iconEl && serviceData.icon) {
            if (serviceData.icon.startsWith('http')) {
                iconEl.innerHTML = `<img src="${serviceData.icon}" alt="${serviceData.name}" />`;
            } else {
                iconEl.innerHTML = serviceData.icon;
            }
        }
    }

    // Auto Refresh
    setupAutoRefresh() {
        const autoRefreshEnabled = localStorage.getItem('autoRefresh') !== 'false';
        const refreshInterval = parseInt(localStorage.getItem('refreshInterval')) || 30000;

        if (autoRefreshEnabled) {
            setInterval(() => {
                this.refreshDashboard();
            }, refreshInterval);
        }
    }

    refreshDashboard() {
        // Only refresh if WebSocket is not connected
        if (!this.websocket || this.websocket.readyState !== WebSocket.OPEN) {
            window.location.reload();
        }
    }

    // Notifications
    setupNotifications() {
        // Request notification permission
        if ('Notification' in window && Notification.permission === 'default') {
            Notification.requestPermission();
        }
    }

    showNotification(message, type = 'info', duration = 5000) {
        // Create toast notification
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.innerHTML = `
            <div class="flex items-center justify-between">
                <span>${message}</span>
                <button class="btn btn-sm" onclick="this.parentElement.parentElement.remove()">
                    ×
                </button>
            </div>
        `;

        document.body.appendChild(toast);

        // Show toast
        setTimeout(() => {
            toast.classList.add('show');
        }, 100);

        // Auto remove
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                if (toast.parentElement) {
                    toast.remove();
                }
            }, 300);
        }, duration);

        // Browser notification
        if ('Notification' in window && Notification.permission === 'granted') {
            if (type === 'error' || type === 'warning') {
                new Notification('CasDash Alert', {
                    body: message,
                    icon: '/static/images/icon.png'
                });
            }
        }
    }

    // Ajax Forms
    async handleAjaxForm(form) {
        const formData = new FormData(form);
        const submitBtn = form.querySelector('button[type="submit"]');
        const originalText = submitBtn.textContent;

        // Show loading state
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner"></span> Loading...';

        try {
            const response = await fetch(form.action, {
                method: form.method,
                body: formData,
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.showNotification(result.message || 'Operation completed successfully', 'success');

                // Redirect if specified
                if (result.redirect) {
                    window.location.href = result.redirect;
                }

                // Reload if specified
                if (result.reload) {
                    window.location.reload();
                }
            } else {
                throw new Error(result.error || 'An error occurred');
            }
        } catch (error) {
            this.showNotification(error.message, 'error');
        } finally {
            // Restore button state
            submitBtn.disabled = false;
            submitBtn.textContent = originalText;
        }
    }

    // Search and Filter
    filterServices(query) {
        const cards = document.querySelectorAll('.service-card');
        const lowerQuery = query.toLowerCase();

        cards.forEach(card => {
            const name = card.querySelector('.service-name')?.textContent.toLowerCase() || '';
            const url = card.querySelector('.service-url')?.textContent.toLowerCase() || '';
            const type = card.dataset.serviceType?.toLowerCase() || '';

            const matches = name.includes(lowerQuery) ||
                          url.includes(lowerQuery) ||
                          type.includes(lowerQuery);

            card.style.display = matches ? 'block' : 'none';
        });
    }

    // Theme Management
    toggleTheme() {
        const currentTheme = localStorage.getItem('theme') || 'dracula';
        const newTheme = currentTheme === 'dracula' ? 'light' : 'dracula';

        localStorage.setItem('theme', newTheme);
        document.documentElement.setAttribute('data-theme', newTheme);

        this.showNotification(`Switched to ${newTheme} theme`, 'info');
    }

    // Utility Functions
    formatTimeAgo(date) {
        const now = new Date();
        const diffInSeconds = Math.floor((now - date) / 1000);

        if (diffInSeconds < 60) {
            return `${diffInSeconds}s ago`;
        } else if (diffInSeconds < 3600) {
            const minutes = Math.floor(diffInSeconds / 60);
            return `${minutes}m ago`;
        } else if (diffInSeconds < 86400) {
            const hours = Math.floor(diffInSeconds / 3600);
            return `${hours}h ago`;
        } else {
            const days = Math.floor(diffInSeconds / 86400);
            return `${days}d ago`;
        }
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';

        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));

        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    formatDuration(milliseconds) {
        const seconds = Math.floor(milliseconds / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (days > 0) {
            return `${days}d ${hours % 24}h`;
        } else if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    }

    // API Helpers
    async apiCall(endpoint, options = {}) {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest'
            }
        };

        const mergedOptions = { ...defaultOptions, ...options };

        try {
            const response = await fetch(`/api/v1${endpoint}`, mergedOptions);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('API call failed:', error);
            throw error;
        }
    }

    // Service Actions
    async checkService(serviceId) {
        try {
            await this.apiCall(`/services/${serviceId}/check`, { method: 'POST' });
            this.showNotification('Service check initiated', 'info');
        } catch (error) {
            this.showNotification(`Failed to check service: ${error.message}`, 'error');
        }
    }

    async toggleMaintenance(serviceId) {
        try {
            await this.apiCall(`/services/${serviceId}/maintenance`, { method: 'POST' });
            this.showNotification('Maintenance mode toggled', 'info');
        } catch (error) {
            this.showNotification(`Failed to toggle maintenance: ${error.message}`, 'error');
        }
    }

    // Chart Helpers (for monitoring pages)
    updateMonitoringData(data) {
        // Update charts if Chart.js is available
        if (typeof Chart !== 'undefined') {
            this.updateResponseTimeChart(data.responseTime);
            this.updateUptimeChart(data.uptime);
        }
    }

    updateResponseTimeChart(data) {
        const chart = Chart.getChart('responseTimeChart');
        if (chart) {
            chart.data.labels.push(new Date().toLocaleTimeString());
            chart.data.datasets[0].data.push(data);

            // Keep only last 50 points
            if (chart.data.labels.length > 50) {
                chart.data.labels.shift();
                chart.data.datasets[0].data.shift();
            }

            chart.update('none');
        }
    }

    updateUptimeChart(data) {
        const chart = Chart.getChart('uptimeChart');
        if (chart) {
            chart.data.datasets[0].data = [data.uptime, 100 - data.uptime];
            chart.update();
        }
    }
}

// Initialize CasDash when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.casdash = new CasDash();
});

// Service Worker Registration (for PWA functionality)
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/static/js/sw.js')
            .then(registration => {
                console.log('ServiceWorker registered: ', registration);
            })
            .catch(registrationError => {
                console.log('ServiceWorker registration failed: ', registrationError);
            });
    });
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = CasDash;
}