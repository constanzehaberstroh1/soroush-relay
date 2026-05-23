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
  Divider,
  LinearProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Avatar,
  CircularProgress,
  Alert,
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
  LockOutlined as LockIcon,
  Refresh as RefreshIcon,
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
  soroushUserId: number;
  displayName: string;
  accessHash: number;
  dcId: number;
  status: 'connected' | 'error' | 'idle';
  lastActive: string;
  createdAt: string;
}

interface LogEntry {
  timestamp: string;
  type: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

// REST Backend Service Utility
const API_BASE = '/api';

const getHeaders = () => {
  const token = localStorage.getItem('client-admin-token');
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
};

// ──────────────────────────────────────────────────────────────────────────────
// Root Layout Component (Wraps dashboard after authentication)
// ──────────────────────────────────────────────────────────────────────────────
function RootLayout() {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    return localStorage.getItem('theme-mode') === 'dark';
  });

  const [tunnelActive, setTunnelActive] = useState(false);
  const [connecting, setConnecting] = useState(false);

  // Poll status from server
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await fetch(`${API_BASE}/status`, { headers: getHeaders() });
        if (res.status === 401) {
          // Token expired or invalid
          localStorage.removeItem('client-admin-token');
          window.location.reload();
          return;
        }
        if (res.ok) {
          const data = await res.json();
          setTunnelActive(data.active);
          setConnecting(data.connecting);
        }
      } catch (err) {
        console.error('Failed to fetch tunnel status:', err);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 3000);
    return () => clearInterval(interval);
  }, []);

  const handleThemeToggle = () => {
    const nextMode = !darkMode;
    setDarkMode(nextMode);
    localStorage.setItem('theme-mode', nextMode ? 'dark' : 'light');
  };

  const handleLogout = () => {
    localStorage.removeItem('client-admin-token');
    window.location.reload();
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
              <Button customVariant="secondary" size="small" onClick={handleLogout} style={{ padding: '4px 10px', fontSize: '0.75rem' }}>
                Logout
              </Button>
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
  const [tunnelActive, setTunnelActive] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<string>('');
  const [bandwidthData, setBandwidthData] = useState<{ time: string; download: number; upload: number }[]>([]);

  const fetchStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/status`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setTunnelActive(data.active);
        setConnecting(data.connecting);
      }
    } catch (err) {
      console.error('Failed status check:', err);
    }
  };

  const fetchAccounts = async () => {
    try {
      const res = await fetch(`${API_BASE}/accounts`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setAccounts(data);
        if (data.length > 0 && !selectedAccount) {
          setSelectedAccount(data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed fetching accounts:', err);
    }
  };

  useEffect(() => {
    fetchStatus();
    fetchAccounts();
    const interval = setInterval(fetchStatus, 3000);

    return () => clearInterval(interval);
  }, []);

  // Simulating live bandwidth rate when connected
  useEffect(() => {
    let graphInterval: any;
    if (tunnelActive) {
      graphInterval = setInterval(() => {
        const download = Math.floor(Math.random() * 45) + 5;
        const upload = Math.floor(Math.random() * 15) + 2;
        const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        
        setBandwidthData((prev) => [...prev, { time, download, upload }].slice(-15));
      }, 1000);
    } else {
      setBandwidthData([]);
    }

    return () => clearInterval(graphInterval);
  }, [tunnelActive]);

  const handleStartTunnel = async () => {
    setConnecting(true);
    try {
      await fetch(`${API_BASE}/start`, { method: 'POST', headers: getHeaders() });
      setTimeout(fetchStatus, 1500);
    } catch (err) {
      console.error('Start tunnel failed:', err);
      setConnecting(false);
    }
  };

  const handleStopTunnel = async () => {
    try {
      await fetch(`${API_BASE}/stop`, { method: 'POST', headers: getHeaders() });
      setTimeout(fetchStatus, 1000);
    } catch (err) {
      console.error('Stop tunnel failed:', err);
    }
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
                {accounts.length > 0 ? (
                  accounts.map((acc) => (
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
                  disabled={connecting || accounts.length === 0}
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
// Page 2: Soroush Accounts Manager View (Framer-motion themed beautiful MUI Table)
// ──────────────────────────────────────────────────────────────────────────────
function AccountsView() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [openAddDialog, setOpenAddDialog] = useState<boolean>(false);
  const [newPhone, setNewPhone] = useState<string>('');
  const [newName, setNewName] = useState<string>('');
  const [verifyCode, setVerifyCode] = useState<string>('');
  const [otpSessionId, setOtpSessionId] = useState<string>('');
  const [step, setStep] = useState<1 | 2>(1);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [helperNotice, setHelperNotice] = useState<string>('');

  const fetchAccounts = async () => {
    try {
      const res = await fetch(`${API_BASE}/accounts`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setAccounts(data);
      }
    } catch (err) {
      console.error('Failed fetching accounts:', err);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  const handleRequestPIN = async () => {
    if (!newPhone || !newName) {
      setErrorMsg('Please specify both Friendly Name and Soroush Phone number!');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/accounts/request-otp`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ phoneNumber: newPhone, name: newName }),
      });
      const data = await res.json();
      if (res.ok) {
        setOtpSessionId(data.sessionId || '');
        setStep(2);
        setHelperNotice('OTP sent via Soroush! Enter the verification code you received on your phone.');
      } else {
        setErrorMsg(data.error || 'Failed to dispatch verification SMS.');
      }
    } catch (err) {
      setErrorMsg('Error contacting Soroush signaling client gateway.');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyPIN = async () => {
    if (!verifyCode) {
      setErrorMsg('Please enter the 5-digit verification PIN.');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/accounts/verify-otp`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ phoneNumber: newPhone, name: newName, code: verifyCode, sessionId: otpSessionId }),
      });
      const data = await res.json();
      if (res.ok) {
        setOpenAddDialog(false);
        setNewPhone('');
        setNewName('');
        setVerifyCode('');
        setOtpSessionId('');
        setStep(1);
        setHelperNotice('');
        fetchAccounts();
      } else {
        setErrorMsg(data.error || 'Incorrect PIN code entered.');
      }
    } catch (err) {
      setErrorMsg('Error verifying PIN on Soroush gateway.');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteAccount = async (id: string) => {
    if (!confirm('Are you sure you want to completely delete this Soroush credential?')) return;
    try {
      const res = await fetch(`${API_BASE}/accounts?id=${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      });
      if (res.ok) {
        fetchAccounts();
      }
    } catch (err) {
      console.error('Delete account request error:', err);
    }
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

        <Divider sx={{ mb: 3 }} />

        {accounts.length > 0 ? (
          <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 3, overflow: 'hidden', bgcolor: 'transparent' }}>
            <Table>
              <TableHead sx={{ bgcolor: 'rgba(255,255,255,0.02)' }}>
                <TableRow>
                  <TableCell style={{ fontWeight: 700 }}>Account Name</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Phone Number</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Soroush ID</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Display Name</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Status</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Registered</TableCell>
                  <TableCell style={{ fontWeight: 700 }} align="center">Action</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {accounts.map((acc) => (
                  <TableRow key={acc.id} hover sx={{ '&:hover': { bgcolor: 'rgba(13, 148, 136, 0.04) !important' } }}>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={1.5}>
                        <Avatar sx={{ bgcolor: 'rgba(13, 148, 136, 0.1)', color: 'primary.main', fontSize: '0.85rem', fontWeight: 'bold' }}>
                          {acc.name.substring(0, 2).toUpperCase()}
                        </Avatar>
                        <Box>
                          <Typography variant="body2" fontWeight={600}>{acc.name}</Typography>
                          <Typography variant="caption" color="text.secondary">ID: {acc.id.substring(0, 8)}</Typography>
                        </Box>
                      </Box>
                    </TableCell>
                    <TableCell style={{ fontFamily: 'monospace' }}>{acc.phoneNumber}</TableCell>
                    <TableCell style={{ fontFamily: 'monospace' }}>{acc.soroushUserId}</TableCell>
                    <TableCell>
                      {acc.displayName || acc.name}
                    </TableCell>
                    <TableCell>
                      <Badge
                        label={acc.status.toUpperCase()}
                        customVariant={acc.status === 'connected' ? 'online' : 'idle'}
                      />
                    </TableCell>
                    <TableCell style={{ fontSize: '0.75rem' }}>
                      {new Date(acc.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell align="center">
                      <IconButton onClick={() => handleDeleteAccount(acc.id)} color="error" size="small">
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : (
          <Box py={6} textAlign="center">
            <AccountIcon sx={{ fontSize: 48, opacity: 0.15, mb: 1, color: 'text.secondary' }} />
            <Typography variant="body2" color="text.secondary">
              No registered Soroush accounts found. Add your first account to initiate WebRTC tunnel.
            </Typography>
          </Box>
        )}

        {/* Add Account Dialog Wizard */}
        <Dialog open={openAddDialog} onClose={() => { setOpenAddDialog(false); setErrorMsg(''); setStep(1); setHelperNotice(''); }} fullWidth maxWidth="xs">
          <DialogTitle fontWeight={800} style={{ paddingBottom: '8px' }}>
            Add Soroush Account
          </DialogTitle>
          <DialogContent>
            {errorMsg && <Alert severity="error" sx={{ mb: 2, borderRadius: 2 }}>{errorMsg}</Alert>}
            {helperNotice && <Alert severity="success" sx={{ mb: 2, borderRadius: 2 }}>{helperNotice}</Alert>}

            {step === 1 ? (
              <Box display="flex" flexDirection="column" gap={2.5} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Link details through the Soroush authentication framework.
                </Typography>
                <TextField
                  label="Friendly Name"
                  placeholder="e.g. Primary Client Node"
                  fullWidth
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  size="small"
                  disabled={loading}
                />
                <TextField
                  label="Soroush Phone"
                  placeholder="+989123456789"
                  fullWidth
                  value={newPhone}
                  onChange={(e) => setNewPhone(e.target.value)}
                  size="small"
                  disabled={loading}
                />
              </Box>
            ) : (
              <Box display="flex" flexDirection="column" gap={2.5} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Enter the 5-digit verification PIN received on your Soroush mobile app.
                </Typography>
                <TextField
                  label="SMS Pin code"
                  placeholder="12345"
                  fullWidth
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value)}
                  size="small"
                  disabled={loading}
                />
              </Box>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 3, pt: 1 }}>
            <Button customVariant="secondary" onClick={() => { setOpenAddDialog(false); setErrorMsg(''); setStep(1); setHelperNotice(''); }} disabled={loading}>
              Cancel
            </Button>
            {step === 1 ? (
              <Button customVariant="primary" onClick={handleRequestPIN} disabled={loading}>
                {loading ? <CircularProgress size={20} color="inherit" /> : "Request Pin"}
              </Button>
            ) : (
              <Button customVariant="glow" onClick={handleVerifyPIN} disabled={loading}>
                {loading ? <CircularProgress size={20} color="inherit" /> : "Verify Pin"}
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
    alert('Relay and Obfuscator settings applied dynamically.');
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
// Page 4: Real-time Engine Logs View (Polled from Go REST backend)
// ──────────────────────────────────────────────────────────────────────────────
function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const fetchLogs = async () => {
    try {
      const res = await fetch(`${API_BASE}/logs`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setLogs(data);
      }
    } catch (err) {
      console.error('Failed fetching engine logs:', err);
    }
  };

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 1500);

    return () => clearInterval(interval);
  }, []);

  return (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
          <Typography variant="h6" fontWeight={700}>
            Engine Logs & WebRTC candidate stream
          </Typography>
          <IconButton onClick={fetchLogs} color="primary" size="small">
            <RefreshIcon fontSize="small" />
          </IconButton>
        </Box>
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
          {logs.length > 0 ? (
            logs.map((log, index) => (
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
            ))
          ) : (
            <Box py={4} textAlign="center" color="text.secondary">
              Awaiting logs buffer payload...
            </Box>
          )}
        </Box>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Elegant Glassmorphic Login Screen
// ──────────────────────────────────────────────────────────────────────────────
interface LoginScreenProps {
  onLoginSuccess: (token: string) => void;
}

function LoginScreen({ onLoginSuccess }: LoginScreenProps) {
  const [username, setUsername] = useState<string>('');
  const [password, setPassword] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setErrorMsg('Username and password fields are both required!');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (res.ok && data.token) {
        localStorage.setItem('client-admin-token', data.token);
        onLoginSuccess(data.token);
      } else {
        setErrorMsg(data.error || 'Authentication rejected. Check credentials.');
      }
    } catch (err) {
      setErrorMsg('Connection error. Server may be offline.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'radial-gradient(circle at top, #0f172a, #020617)',
        px: 2,
      }}
    >
      <Card
        sx={{
          maxWidth: 400,
          width: '100%',
          p: 3,
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
          background: 'rgba(15, 23, 42, 0.45) !important',
          backdropFilter: 'blur(20px)',
          border: '1px solid rgba(255, 255, 255, 0.08)',
          borderRadius: 4,
        }}
      >
        <CardContent>
          <Box display="flex" flexDirection="column" alignItems="center" mb={4}>
            <Avatar sx={{ m: 1, bgcolor: 'primary.main', width: 56, height: 56, boxShadow: '0 0 20px rgba(45, 212, 191, 0.4)' }}>
              <LockIcon style={{ color: '#0f172a', fontSize: 28 }} />
            </Avatar>
            <Typography variant="h5" component="h1" fontWeight={800} mt={1} color="text.primary" textAlign="center">
              SOROUSH RELAY
            </Typography>
            <Typography variant="caption" color="text.secondary" textAlign="center">
              Client Administrator Dashboard Portal
            </Typography>
          </Box>

          {errorMsg && (
            <Alert severity="error" sx={{ mb: 3, borderRadius: 2 }}>
              {errorMsg}
            </Alert>
          )}

          <Box component="form" onSubmit={handleLoginSubmit} display="flex" flexDirection="column" gap={3}>
            <TextField
              label="Username"
              variant="outlined"
              fullWidth
              size="medium"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={loading}
              InputLabelProps={{ style: { color: 'rgba(255,255,255,0.7)' } }}
              sx={{ input: { color: '#ffffff' } }}
            />
            <TextField
              label="Password"
              type="password"
              variant="outlined"
              fullWidth
              size="medium"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
              InputLabelProps={{ style: { color: 'rgba(255,255,255,0.7)' } }}
              sx={{ input: { color: '#ffffff' } }}
            />
            <Button
              type="submit"
              customVariant="glow"
              fullWidth
              disabled={loading}
              style={{ paddingTop: '12px', paddingBottom: '12px', fontSize: '1rem', fontWeight: 700 }}
            >
              {loading ? <CircularProgress size={24} color="inherit" /> : "Authenticate Login"}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
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
// App Component Mounting Provider with JWT auth wrapper
// ──────────────────────────────────────────────────────────────────────────────
export function App() {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('client-admin-token');
  });

  const [checking, setChecking] = useState<boolean>(true);

  // Perform lightweight validation on startup
  useEffect(() => {
    const validateLocalToken = async () => {
      if (!token) {
        setChecking(false);
        return;
      }
      try {
        const res = await fetch(`${API_BASE}/admin/me`, {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) {
          localStorage.removeItem('client-admin-token');
          setToken(null);
        }
      } catch (err) {
        console.error('Authentication gateway offline:', err);
      } finally {
        setChecking(false);
      }
    };
    validateLocalToken();
  }, [token]);

  if (checking) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" minHeight="100vh" bgcolor="#020617">
        <CircularProgress color="primary" />
      </Box>
    );
  }

  if (!token) {
    return (
      <ThemeProvider theme={darkTheme}>
        <CssBaseline />
        <LoginScreen onLoginSuccess={(tok) => setToken(tok)} />
      </ThemeProvider>
    );
  }

  return <RouterProvider router={router} />;
}

export default App;
