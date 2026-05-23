import { useState, useEffect } from 'react';
import {
  ThemeProvider,
  CssBaseline,
  Box,
  Container,
  Grid,
  Paper,
  Typography,
  TextField,
  Card,
  CardContent,
  IconButton,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  LinearProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Divider,
} from '@mui/material';
import {
  Brightness4 as DarkModeIcon,
  Brightness7 as LightModeIcon,
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  NetworkCheck as NetworkIcon,
  SettingsInputComponent as ConnectionIcon,
  AccountCircle as AccountIcon,
  VpnLock as TunnelIcon,
  Dashboard as DashboardIcon,
  HistoryToggleOff as LogsIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { lightTheme, darkTheme } from './theme';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import {
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
  Link,
  Outlet,
} from '@tanstack/react-router';

// Atomic Imports
import { Button } from './components/atoms/Button';
import { Badge } from './components/atoms/Badge';

// ──────────────────────────────────────────────────────────────────────────────
// Interfaces & Types
// ──────────────────────────────────────────────────────────────────────────────
interface Account {
  id: string;
  phoneNumber: string;
  name: string;
  status: 'connected' | 'error' | 'idle';
  lastActive: string;
}

interface LogEntry {
  timestamp: string;
  type: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

// ──────────────────────────────────────────────────────────────────────────────
// Global Core State / Context Simulation
// ──────────────────────────────────────────────────────────────────────────────
let globalTunnelActive = false;
let globalConnecting = false;
let globalAccounts: Account[] = [];
let globalLogs: LogEntry[] = [
  { timestamp: new Date().toLocaleTimeString(), type: 'info', message: 'Soroush WebRTC engine initialized.' },
  { timestamp: new Date().toLocaleTimeString(), type: 'success', message: 'Ready to establish P2P voice call channel wrapper' },
];
let globalBandwidthData: { time: string; download: number; upload: number }[] = [];
let listeners: (() => void)[] = [];

const subscribe = (cb: () => void) => {
  listeners.push(cb);
  return () => {
    listeners = listeners.filter((l) => l !== cb);
  };
};

const notify = () => {
  listeners.forEach((cb) => cb());
};

const setTunnelState = (active: boolean, conn: boolean) => {
  globalTunnelActive = active;
  globalConnecting = conn;
  notify();
};

const addLog = (message: string, type: 'info' | 'success' | 'warn' | 'error' = 'info') => {
  const time = new Date().toLocaleTimeString();
  globalLogs = [{ timestamp: time, type, message }, ...globalLogs].slice(0, 100);
  notify();
};

// ──────────────────────────────────────────────────────────────────────────────
// Root Layout Component
// ──────────────────────────────────────────────────────────────────────────────
function RootLayout() {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    return localStorage.getItem('theme-mode') === 'dark';
  });

  const [tunnelActive, setTunnelActive] = useState(globalTunnelActive);
  const [connecting, setConnecting] = useState(globalConnecting);

  useEffect(() => {
    return subscribe(() => {
      setTunnelActive(globalTunnelActive);
      setConnecting(globalConnecting);
    });
  }, []);

  const handleThemeToggle = () => {
    const nextMode = !darkMode;
    setDarkMode(nextMode);
    localStorage.setItem('theme-mode', nextMode ? 'dark' : 'light');
  };

  const theme = darkMode ? darkTheme : lightTheme;

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', bgcolor: 'background.default' }}>
        {/* Header App Bar */}
        <Paper elevation={0} sx={{ borderBottom: 1, borderColor: 'divider', py: 2.5, px: 4, borderRadius: 0 }}>
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Box display="flex" alignItems="center" gap={2}>
              <TunnelIcon color="primary" sx={{ fontSize: 28 }} className={tunnelActive ? 'pulsing-glow' : ''} />
              <Typography variant="h6" color="text.primary" fontWeight={800} style={{ letterSpacing: '-0.01em' }}>
                SOROUSH RELAY
                <Box component="span" sx={{ color: 'primary.main', ml: 1, fontWeight: 400, fontSize: '0.85em' }}>
                  Client Dashboard
                </Box>
              </Typography>
            </Box>
            
            <Box display="flex" alignItems="center" gap={2}>
              <Badge
                label={tunnelActive ? "CONNECTED" : connecting ? "CONNECTING..." : "DISCONNECTED"}
                customVariant={tunnelActive ? "online" : connecting ? "connecting" : "offline"}
              />
              <IconButton onClick={handleThemeToggle} color="inherit">
                {darkMode ? <LightModeIcon /> : <DarkModeIcon />}
              </IconButton>
            </Box>
          </Box>
        </Paper>

        {/* Dynamic Navigation Bar */}
        <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper', px: 4, py: 1 }}>
          <Box display="flex" gap={1}>
            <Link
              to="/"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(13, 148, 136, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <DashboardIcon fontSize="small" /> Dashboard
            </Link>
            <Link
              to="/accounts"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(13, 148, 136, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <AccountIcon fontSize="small" /> Soroush Accounts
            </Link>
            <Link
              to="/settings"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(13, 148, 136, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <SettingsIcon fontSize="small" /> Settings
            </Link>
            <Link
              to="/logs"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(13, 148, 136, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <LogsIcon fontSize="small" /> Logs Console
            </Link>
          </Box>
        </Box>

        {/* Main Content Router Outlet */}
        <Container maxWidth="xl" sx={{ mt: 4, mb: 4, flexGrow: 1 }}>
          <Outlet />
        </Container>
      </Box>
    </ThemeProvider>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 1: Dashboard Home View
// ──────────────────────────────────────────────────────────────────────────────
function DashboardView() {
  const [tunnelActive, setTunnelActive] = useState(globalTunnelActive);
  const [connecting, setConnecting] = useState(globalConnecting);
  const [selectedAccount, setSelectedAccount] = useState<string>('acc-1');
  const [bandwidthData, setBandwidthData] = useState(globalBandwidthData);

  useEffect(() => {
    const unsub = subscribe(() => {
      setTunnelActive(globalTunnelActive);
      setConnecting(globalConnecting);
    });

    let graphInterval: any;
    if (tunnelActive) {
      graphInterval = setInterval(() => {
        const download = Math.floor(Math.random() * 45) + 5;
        const upload = Math.floor(Math.random() * 15) + 2;
        const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        
        globalBandwidthData = [...globalBandwidthData, { time, download, upload }].slice(-15);
        setBandwidthData(globalBandwidthData);
      }, 1000);
    } else {
      globalBandwidthData = [];
      setBandwidthData([]);
    }

    return () => {
      unsub();
      clearInterval(graphInterval);
    };
  }, [tunnelActive]);

  const handleStartTunnel = () => {
    setTunnelState(false, true);
    addLog('Connecting Soroush API WebSocket handshake...', 'info');
    
    setTimeout(() => {
      addLog('apiws WebSocket established successfully', 'success');
      addLog('Session initiation sent with keep-alive heartbeat interval = 30s', 'info');
      
      setTimeout(() => {
        addLog('Received WebRTC SDP Offer from Soroush Server', 'success');
        addLog('Local WebRTC ICE candidates generated and sent to signaling channel', 'info');
        
        setTimeout(() => {
          addLog('SOCKS5 Proxy interface listening on port 4046', 'success');
          setTunnelState(true, false);
          addLog('Traffic successfully obfuscated as Soroush voice call payload!', 'success');
        }, 1000);
      }, 1000);
    }, 1500);
  };

  const handleStopTunnel = () => {
    addLog('Closing WebRTC data channel...', 'info');
    setTunnelState(false, false);
    addLog('Soroush WebRTC Tunnel stopped safely.', 'warn');
  };

  return (
    <Grid container spacing={3}>
      {/* Control Panel */}
      <Grid item xs={12} md={4}>
        <Card>
          <CardContent>
            <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
              <Typography variant="h6" fontWeight={700}>
                Tunnel Status
              </Typography>
              <ConnectionIcon color="secondary" />
            </Box>
            
            <Divider sx={{ my: 2 }} />

            <Box display="flex" flexDirection="column" gap={2}>
              {connecting && <LinearProgress color="primary" />}
              
              <Typography variant="body2" color="text.secondary">
                Route all your traffic through the Soroush messenger WebRTC data tunnel.
              </Typography>

              <Paper variant="outlined" sx={{ p: 2, bgcolor: 'background.default', borderRadius: 2 }}>
                <Grid container spacing={1}>
                  <Grid item xs={6}>
                    <Typography variant="caption" color="text.secondary">Local Host</Typography>
                    <Typography variant="body2" fontWeight="bold">127.0.0.1</Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="caption" color="text.secondary">SOCKS5 Port</Typography>
                    <Typography variant="body2" fontWeight="bold">4046</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="caption" color="text.secondary">Obfuscation Protocol</Typography>
                    <Typography variant="body2" color="primary.main" fontWeight="bold">Soroush Voice Call Masquerade</Typography>
                  </Grid>
                </Grid>
              </Paper>

              <TextField
                select
                label="Signaling Soroush Account"
                value={selectedAccount}
                onChange={(e) => setSelectedAccount(e.target.value)}
                SelectProps={{ native: true }}
                fullWidth
                size="small"
                disabled={tunnelActive || connecting}
              >
                {globalAccounts.length > 0 ? (
                  globalAccounts.map((acc) => (
                    <option key={acc.id} value={acc.id}>
                      {acc.name} ({acc.phoneNumber})
                    </option>
                  ))
                ) : (
                  <option value="">No accounts linked yet</option>
                )}
              </TextField>

              {!tunnelActive ? (
                <Button
                  customVariant="glow"
                  startIcon={<StartIcon />}
                  onClick={handleStartTunnel}
                  disabled={connecting || globalAccounts.length === 0}
                  fullWidth
                >
                  {connecting ? "Connecting..." : "Start Soroush Tunnel"}
                </Button>
              ) : (
                <Button
                  customVariant="danger"
                  startIcon={<StopIcon />}
                  onClick={handleStopTunnel}
                  fullWidth
                >
                  Disconnect Tunnel
                </Button>
              )}
            </Box>
          </CardContent>
        </Card>
      </Grid>

      {/* Graph Area */}
      <Grid item xs={12} md={8}>
        <Card>
          <CardContent sx={{ height: 360, display: 'flex', flexDirection: 'column' }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6" fontWeight={700}>
                Traffic Rate
              </Typography>
              <NetworkIcon color="primary" />
            </Box>
            
            <Box sx={{ flexGrow: 1, minHeight: 0 }}>
              {tunnelActive && bandwidthData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={bandwidthData}>
                    <defs>
                      <linearGradient id="colorDownload" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#0d9488" stopOpacity={0.2}/>
                        <stop offset="95%" stopColor="#0d9488" stopOpacity={0.0}/>
                      </linearGradient>
                      <linearGradient id="colorUpload" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#6366f1" stopOpacity={0.2}/>
                        <stop offset="95%" stopColor="#6366f1" stopOpacity={0.0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.06)" />
                    <XAxis dataKey="time" stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <YAxis stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <Tooltip contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155' }} />
                    <Area type="monotone" dataKey="download" stroke="#0d9488" fillOpacity={1} fill="url(#colorDownload)" name="Download (MB/s)" />
                    <Area type="monotone" dataKey="upload" stroke="#6366f1" fillOpacity={1} fill="url(#colorUpload)" name="Upload (MB/s)" />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100%" gap={1}>
                  <NetworkIcon color="disabled" sx={{ fontSize: 48, opacity: 0.2 }} />
                  <Typography variant="body2" color="text.secondary">
                    WebRTC data channel is currently idle. Connect to visualize tunnel bandwidth.
                  </Typography>
                </Box>
              )}
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 2: Soroush Accounts Manager View
// ──────────────────────────────────────────────────────────────────────────────
function AccountsView() {
  const [accounts, setAccounts] = useState<Account[]>(globalAccounts);
  const [openAddDialog, setOpenAddDialog] = useState<boolean>(false);
  const [newPhone, setNewPhone] = useState<string>('');
  const [newName, setNewName] = useState<string>('');
  const [verifyCode, setVerifyCode] = useState<string>('');
  const [step, setStep] = useState<1 | 2>(1);

  const handleAddAccount = () => {
    if (!newPhone || !newName) return;
    setStep(2);
    addLog(`Sending SMS verification request to ${newPhone} over Soroush Web gateway...`, 'info');
  };

  const handleVerifyAccount = () => {
    if (!verifyCode) return;
    const newAcc: Account = {
      id: `acc-${Date.now()}`,
      phoneNumber: newPhone,
      name: newName,
      status: 'idle',
      lastActive: 'Just registered',
    };
    globalAccounts = [...globalAccounts, newAcc];
    setAccounts(globalAccounts);
    addLog(`Account ${newPhone} verified and linked to Soroush WebSocket pool.`, 'success');
    
    setOpenAddDialog(false);
    setNewPhone('');
    setNewName('');
    setVerifyCode('');
    setStep(1);
  };

  const handleDeleteAccount = (id: string, phone: string) => {
    globalAccounts = globalAccounts.filter((acc) => acc.id !== id);
    setAccounts(globalAccounts);
    addLog(`Account ${phone} removed from Soroush credential library.`, 'warn');
  };

  return (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
          <Typography variant="h6" fontWeight={700}>
            Registered Soroush Credentials
          </Typography>
          <Button customVariant="primary" startIcon={<AddIcon />} onClick={() => setOpenAddDialog(true)}>
            Add Account
          </Button>
        </Box>
        <Typography variant="body2" color="text.secondary" mb={3}>
          Link accounts using your registered Soroush phone number to build signaling paths dynamically.
        </Typography>

        <Divider sx={{ mb: 2 }} />

        {accounts.length > 0 ? (
          <List>
            {accounts.map((account) => (
              <ListItem key={account.id} sx={{ px: 2, borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
                <Box display="flex" alignItems="center" gap={2} width="100%">
                  <AccountIcon color={account.status === 'connected' ? 'primary' : 'disabled'} />
                  <ListItemText
                    primary={account.name}
                    secondary={
                      <Box component="span" display="flex" flexDirection="column">
                        <span>{account.phoneNumber}</span>
                        <Box component="span" display="flex" gap={1.5} alignItems="center" mt={0.5}>
                          <Badge
                            label={account.status.toUpperCase()}
                            customVariant={account.status === 'connected' ? 'online' : 'idle'}
                          />
                          <Typography variant="caption" color="text.secondary">
                            Last active: {account.lastActive}
                          </Typography>
                        </Box>
                      </Box>
                    }
                  />
                  <ListItemSecondaryAction>
                    <IconButton
                      edge="end"
                      onClick={() => handleDeleteAccount(account.id, account.phoneNumber)}
                      disabled={account.status === 'connected' && globalTunnelActive}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </ListItemSecondaryAction>
                </Box>
              </ListItem>
            ))}
          </List>
        ) : (
          <Box py={6} textAlign="center">
            <Typography variant="body2" color="text.secondary">
              No registered Soroush accounts found. Add your first account to initiate WebRTC tunnel.
            </Typography>
          </Box>
        )}

        {/* Add Account Dialog */}
        <Dialog open={openAddDialog} onClose={() => setOpenAddDialog(false)} fullWidth maxWidth="xs">
          <DialogTitle fontWeight={800}>
            Add Soroush Account
          </DialogTitle>
          <DialogContent>
            {step === 1 ? (
              <Box display="flex" flexDirection="column" gap={2} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Register via your Soroush messaging account credentials.
                </Typography>
                <TextField
                  label="Friendly Name"
                  placeholder="e.g. Primary Node"
                  fullWidth
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  size="small"
                />
                <TextField
                  label="Soroush Phone"
                  placeholder="+989123456789"
                  fullWidth
                  value={newPhone}
                  onChange={(e) => setNewPhone(e.target.value)}
                  size="small"
                />
              </Box>
            ) : (
              <Box display="flex" flexDirection="column" gap={2} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Enter the verification code received on your Soroush mobile app.
                </Typography>
                <TextField
                  label="SMS Pin"
                  placeholder="12345"
                  fullWidth
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value)}
                  size="small"
                />
              </Box>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 3 }}>
            <Button customVariant="secondary" onClick={() => setOpenAddDialog(false)}>Cancel</Button>
            {step === 1 ? (
              <Button customVariant="primary" onClick={handleAddAccount}>
                Request Pin
              </Button>
            ) : (
              <Button customVariant="glow" onClick={handleVerifyAccount}>
                Verify Pin
              </Button>
            )}
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 3: Tunnel Config / Settings View
// ──────────────────────────────────────────────────────────────────────────────
function SettingsView() {
  const [socksPort, setSocksPort] = useState<number>(4046);
  const [exitIP, setExitIP] = useState<string>('127.0.0.1');
  const [exitPort, setExitPort] = useState<number>(8080);
  const [obfuscation, setObfuscation] = useState<string>('voice');

  const handleSave = () => {
    addLog(`System parameters saved. Local proxy port: ${socksPort}, remote gateway target: ${exitIP}:${exitPort}`, 'success');
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" fontWeight={700} mb={1}>
          Relay Config & Routing Settings
        </Typography>
        <Typography variant="body2" color="text.secondary" mb={3}>
          Configure signaling targets, local ports, and WebRTC masquerade algorithms.
        </Typography>

        <Divider sx={{ mb: 3 }} />

        <Grid container spacing={3}>
          <Grid item xs={12} sm={6}>
            <TextField
              label="Local SOCKS5 Listener Port"
              type="number"
              value={socksPort}
              onChange={(e) => setSocksPort(Number(e.target.value))}
              fullWidth
              size="small"
            />
          </Grid>
          <Grid item xs={12} sm={6}>
            <TextField
              label="Soroush Masquerading Type"
              select
              value={obfuscation}
              onChange={(e) => setObfuscation(e.target.value)}
              SelectProps={{ native: true }}
              fullWidth
              size="small"
            >
              <option value="voice">Audio Voice Calling emulation (Highly Resilient)</option>
              <option value="video">H.264 Video Chat simulation</option>
              <option value="files">Standard file transfers channel</option>
            </TextField>
          </Grid>

          <Grid item xs={12} sm={8}>
            <TextField
              label="Target Soroush Server Exit Node (IP / Hostname)"
              value={exitIP}
              onChange={(e) => setExitIP(e.target.value)}
              fullWidth
              size="small"
            />
          </Grid>
          <Grid item xs={12} sm={4}>
            <TextField
              label="Target Service Port"
              type="number"
              value={exitPort}
              onChange={(e) => setExitPort(Number(e.target.value))}
              fullWidth
              size="small"
            />
          </Grid>

          <Grid item xs={12}>
            <Button customVariant="primary" startIcon={<StartIcon />} onClick={handleSave}>
              Apply Specifications
            </Button>
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 4: Real-time Engine Logs View
// ──────────────────────────────────────────────────────────────────────────────
function LogsView() {
  const [logs, setLogs] = useState(globalLogs);

  useEffect(() => {
    return subscribe(() => {
      setLogs(globalLogs);
    });
  }, []);

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" fontWeight={700} mb={2}>
          Engine Logs & WebRTC candidate stream
        </Typography>
        <Box
          sx={{
            height: 480,
            bgcolor: '#020617',
            borderRadius: 3,
            p: 3,
            overflowY: 'auto',
            fontFamily: 'Courier New, monospace',
            fontSize: '0.85rem',
            display: 'flex',
            flexDirection: 'column',
            gap: 1,
            boxShadow: 'inset 0 2px 10px rgba(0,0,0,0.8)',
          }}
        >
          {logs.map((log, index) => (
            <Box key={index} display="flex" gap={1.5}>
              <Box sx={{ color: '#64748b', userSelect: 'none' }}>
                [{log.timestamp}]
              </Box>
              <Box
                sx={{
                  color:
                    log.type === 'success'
                      ? '#10b981'
                      : log.type === 'warn'
                      ? '#f59e0b'
                      : log.type === 'error'
                      ? '#ef4444'
                      : '#94a3b8',
                }}
              >
                {log.message}
              </Box>
            </Box>
          ))}
        </Box>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// TanStack Router Definition (TypeScript Compliant Code Route Tree)
// ──────────────────────────────────────────────────────────────────────────────
const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardView,
});

const accountsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/accounts',
  component: AccountsView,
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: SettingsView,
});

const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/logs',
  component: LogsView,
});

const routeTree = rootRoute.addChildren([indexRoute, accountsRoute, settingsRoute, logsRoute]);

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// App Component Mounting Provider
// ──────────────────────────────────────────────────────────────────────────────
export function App() {
  return <RouterProvider router={router} />;
}

export default App;
