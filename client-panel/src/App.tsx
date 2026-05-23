import { useState, useEffect, useRef } from 'react';
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
  Stepper,
  Step,
  StepLabel,
  Chip,
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
  Groups as GroupIcon,
  Science as ScienceIcon,
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

  // Server connectivity test
  const [serverUrl, setServerUrl] = useState<string>(() => {
    return localStorage.getItem('server-exit-url') || 'https://app-25d61cc9-bc8f-4d35-8fd8-c1a24ad4abb7.cleverapps.io';
  });
  const [testResult, setTestResult] = useState<any>(null);
  const [testing, setTesting] = useState(false);

  // Group Bus Config
  const [groupChatId, setGroupChatId] = useState<string>('');
  const [groupPsk, setGroupPsk] = useState<string>('');
  const [groupSaving, setGroupSaving] = useState(false);
  const [groupSaved, setGroupSaved] = useState(false);

  // Group Picker Modal state
  const [groupPickerOpen, setGroupPickerOpen] = useState(false);
  const [groupPickerLoading, setGroupPickerLoading] = useState(false);
  const [groupPickerError, setGroupPickerError] = useState('');
  const [groupPickerSearch, setGroupPickerSearch] = useState('');
  const [groupList, setGroupList] = useState<{ id: number; title: string; type: string; membersCount: number }[]>([]);

  // Tunnel Test
  const [tunnelTesting, setTunnelTesting] = useState(false);
  const [tunnelTestResult, setTunnelTestResult] = useState<any>(null);
  const [tunnelTestActiveStep, setTunnelTestActiveStep] = useState(-1);

  // Load group config on mount
  useEffect(() => {
    const loadGroupConfig = async () => {
      try {
        const res = await fetch(`${API_BASE}/tunnel/config`, { headers: getHeaders() });
        if (res.ok) {
          const data = await res.json();
          if (data.groupChatId) setGroupChatId(String(data.groupChatId));
          if (data.psk) setGroupPsk(data.psk);
        }
      } catch { /* ignore */ }
    };
    loadGroupConfig();
  }, []);

  const handleSave = () => {
    localStorage.setItem('server-exit-url', serverUrl);
    alert('Settings saved.');
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setTestResult(null);
    localStorage.setItem('server-exit-url', serverUrl);
    try {
      const res = await fetch(`${API_BASE}/test-server`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ serverUrl }),
      });
      if (res.ok) {
        const data = await res.json();
        setTestResult(data);
      } else {
        setTestResult({ success: false, error: 'Client backend returned an error' });
      }
    } catch (err) {
      setTestResult({ success: false, error: 'Failed to reach client backend' });
    } finally {
      setTesting(false);
    }
  };

  const handleSaveGroupConfig = async () => {
    setGroupSaving(true);
    setGroupSaved(false);
    try {
      const res = await fetch(`${API_BASE}/tunnel/config`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          groupChatId: Number(groupChatId),
          psk: groupPsk,
          dispatcherUserId: 0,
          dispatcherAccessHash: 0,
        }),
      });
      if (res.ok) setGroupSaved(true);
    } catch { /* ignore */ }
    finally { setGroupSaving(false); }
  };

  const handleOpenGroupPicker = async () => {
    setGroupPickerOpen(true);
    setGroupPickerLoading(true);
    setGroupPickerError('');
    setGroupList([]);
    try {
      const res = await fetch(`${API_BASE}/groups/list`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setGroupList(data || []);
      } else {
        const err = await res.json();
        setGroupPickerError(err.error || 'Failed to fetch groups.');
      }
    } catch {
      setGroupPickerError('Connection error. Make sure you have a connected Soroush account.');
    } finally {
      setGroupPickerLoading(false);
    }
  };

  const handleSelectGroup = (group: { id: number; title: string }) => {
    setGroupChatId(String(group.id));
    setGroupSaved(false);
    setGroupPickerOpen(false);
  };

  const tunnelTestStepNames = ['MTProto Connect', 'Group DISCOVER', 'WebRTC Call', 'Ping / Pong'];
  const tunnelTestStepKeys = ['mtproto_connect', 'group_discover', 'webrtc_call', 'ping_pong'];

  const handleTunnelTest = async () => {
    setTunnelTesting(true);
    setTunnelTestResult(null);
    setTunnelTestActiveStep(0);
    try {
      const res = await fetch(`${API_BASE}/tunnel/test`, {
        method: 'POST',
        headers: getHeaders(),
      });
      if (res.ok) {
        const data = await res.json();
        setTunnelTestResult(data);
        // Find the last completed step
        const lastIdx = (data.steps || []).findIndex((s: any) => s.status === 'fail' || s.status === 'skip');
        setTunnelTestActiveStep(lastIdx === -1 ? data.steps.length : lastIdx);
      } else {
        setTunnelTestResult({ success: false, steps: [{ name: 'mtproto_connect', status: 'fail', detail: 'Backend error' }] });
        setTunnelTestActiveStep(0);
      }
    } catch {
      setTunnelTestResult({ success: false, steps: [{ name: 'mtproto_connect', status: 'fail', detail: 'Network error' }] });
      setTunnelTestActiveStep(0);
    } finally {
      setTunnelTesting(false);
    }
  };

  const getStepStatus = (stepKey: string) => {
    if (!tunnelTestResult?.steps) return undefined;
    return tunnelTestResult.steps.find((s: any) => s.name === stepKey);
  };

  return (
    <Box display="flex" flexDirection="column" gap={3}>
      {/* Server Connectivity Test Card */}
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
            <Typography variant="h6" fontWeight={700}>
              Server Connection Test
            </Typography>
            <NetworkIcon color="secondary" />
          </Box>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Verify that this client machine can reach the Clever Cloud exit node server over HTTPS.
          </Typography>

          <Divider sx={{ mb: 3 }} />

          <Grid container spacing={2} alignItems="flex-end">
            <Grid item xs={12} sm={8}>
              <TextField
                label="Server Exit Node URL"
                placeholder="https://your-app.cleverapps.io"
                value={serverUrl}
                onChange={(e) => setServerUrl(e.target.value)}
                fullWidth
                size="small"
                disabled={testing}
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <Button
                customVariant="glow"
                onClick={handleTestConnection}
                disabled={testing || !serverUrl}
                fullWidth
                startIcon={testing ? <CircularProgress size={18} color="inherit" /> : <RefreshIcon />}
              >
                {testing ? 'Testing...' : 'Test Connection'}
              </Button>
            </Grid>
          </Grid>

          {testResult && (
            <Box mt={3}>
              {testResult.success ? (
                <Alert
                  severity="success"
                  sx={{ borderRadius: 2 }}
                  icon={<NetworkIcon />}
                >
                  <Box>
                    <Typography variant="body2" fontWeight={700}>
                      ✅ Server is reachable!
                    </Typography>
                    <Typography variant="caption" component="div" sx={{ mt: 0.5 }}>
                      Latency: <strong>{testResult.latencyMs}ms</strong>
                      {testResult.server?.version && (
                        <> · Version: <strong>{testResult.server.version}</strong></>
                      )}
                      {testResult.server?.timestamp && (
                        <> · Server Time: {new Date(testResult.server.timestamp).toLocaleString()}</>
                      )}
                    </Typography>
                  </Box>
                </Alert>
              ) : (
                <Alert severity="error" sx={{ borderRadius: 2 }}>
                  <Box>
                    <Typography variant="body2" fontWeight={700}>
                      ❌ Connection Failed
                    </Typography>
                    <Typography variant="caption" component="div" sx={{ mt: 0.5, wordBreak: 'break-all' }}>
                      {testResult.error}
                      {testResult.latencyMs > 0 && <> (after {testResult.latencyMs}ms)</>}
                    </Typography>
                  </Box>
                </Alert>
              )}
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Group Bus Configuration Card */}
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
            <Typography variant="h6" fontWeight={700}>
              Group Bus Configuration
            </Typography>
            <GroupIcon color="secondary" />
          </Box>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Configure the "My lovely family" Soroush group as the pub/sub message bus for discovery and signaling.
          </Typography>

          <Divider sx={{ mb: 3 }} />

          <Grid container spacing={2} alignItems="flex-end">
            <Grid item xs={12} sm={6}>
              <Box display="flex" gap={1} alignItems="flex-start">
                <TextField
                  label="Group Chat ID"
                  placeholder="e.g. 277900000529092119"
                  value={groupChatId}
                  onChange={(e) => { setGroupChatId(e.target.value); setGroupSaved(false); }}
                  fullWidth
                  size="small"
                  disabled={groupSaving}
                  helperText="The numeric chat_id of your Soroush group"
                />
                <Button
                  customVariant="secondary"
                  size="small"
                  onClick={handleOpenGroupPicker}
                  disabled={groupSaving}
                  style={{ minWidth: 0, padding: '6px 12px', whiteSpace: 'nowrap', marginTop: '1px' }}
                  startIcon={<GroupIcon style={{ fontSize: 16 }} />}
                >
                  Browse
                </Button>
              </Box>
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Pre-Shared Key (PSK)"
                placeholder="Leave empty for default"
                value={groupPsk}
                onChange={(e) => { setGroupPsk(e.target.value); setGroupSaved(false); }}
                fullWidth
                size="small"
                disabled={groupSaving}
                helperText="AES-256 encryption key for stealth group messages"
              />
            </Grid>
            <Grid item xs={12}>
              <Button
                customVariant="primary"
                onClick={handleSaveGroupConfig}
                disabled={groupSaving || !groupChatId}
                startIcon={groupSaving ? <CircularProgress size={18} color="inherit" /> : <StartIcon />}
              >
                {groupSaved ? '✅ Saved!' : 'Save Group Config'}
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Soroush Tunnel Test Card */}
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
            <Typography variant="h6" fontWeight={700}>
              Soroush Tunnel Test
            </Typography>
            <ScienceIcon color="secondary" />
          </Box>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Run a full end-to-end connectivity test: MTProto → Group Discovery → WebRTC Call → Data Channel Ping/Pong.
          </Typography>

          <Divider sx={{ mb: 3 }} />

          {/* Stepper */}
          <Stepper activeStep={tunnelTestActiveStep} alternativeLabel sx={{ mb: 3 }}>
            {tunnelTestStepNames.map((label, index) => {
              const stepData = getStepStatus(tunnelTestStepKeys[index]);
              const isError = stepData?.status === 'fail';
              const isSkip = stepData?.status === 'skip';
              return (
                <Step key={label} completed={stepData?.status === 'pass'}>
                  <StepLabel
                    error={isError}
                    optional={isSkip ? <Typography variant="caption" color="text.secondary">skipped</Typography> : undefined}
                  >
                    {label}
                  </StepLabel>
                </Step>
              );
            })}
          </Stepper>

          {/* Step Results */}
          {tunnelTestResult?.steps && (
            <Box display="flex" flexDirection="column" gap={1} mb={3}>
              {tunnelTestResult.steps.map((step: any, idx: number) => (
                <Box
                  key={idx}
                  display="flex"
                  alignItems="center"
                  gap={1.5}
                  sx={{
                    p: 1.5,
                    borderRadius: 2,
                    bgcolor: step.status === 'pass' ? 'rgba(46,125,50,0.08)' :
                             step.status === 'fail' ? 'rgba(211,47,47,0.08)' : 'rgba(128,128,128,0.06)',
                  }}
                >
                  <Chip
                    label={step.status.toUpperCase()}
                    size="small"
                    color={step.status === 'pass' ? 'success' : step.status === 'fail' ? 'error' : 'default'}
                    sx={{ fontWeight: 700, minWidth: 60 }}
                  />
                  <Typography variant="body2" fontWeight={600} sx={{ minWidth: 140 }}>
                    {tunnelTestStepNames[idx]}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ flex: 1 }}>
                    {step.detail}
                  </Typography>
                  {step.latencyMs > 0 && (
                    <Chip label={`${step.latencyMs}ms`} size="small" variant="outlined" />
                  )}
                </Box>
              ))}
            </Box>
          )}

          {tunnelTestResult && (
            <Alert
              severity={tunnelTestResult.success ? 'success' : 'error'}
              sx={{ borderRadius: 2, mb: 2 }}
            >
              {tunnelTestResult.success
                ? `✅ Tunnel test passed! Total: ${tunnelTestResult.overallLatencyMs}ms`
                : '❌ Tunnel test failed. Check step details above.'}
            </Alert>
          )}

          <Button
            customVariant="glow"
            onClick={handleTunnelTest}
            disabled={tunnelTesting || !groupChatId}
            fullWidth
            startIcon={tunnelTesting ? <CircularProgress size={18} color="inherit" /> : <ScienceIcon />}
          >
            {tunnelTesting ? 'Running Test...' : '🔬 Test Soroush Tunnel'}
          </Button>
        </CardContent>
      </Card>

      {/* Relay Settings Card */}
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
      {/* Group Picker Modal */}
      <Dialog
        open={groupPickerOpen}
        onClose={() => setGroupPickerOpen(false)}
        fullWidth
        maxWidth="sm"
        PaperProps={{
          sx: {
            bgcolor: '#0f172a',
            border: '1px solid rgba(255,255,255,0.08)',
            borderRadius: 3,
          }
        }}
      >
        <DialogTitle fontWeight={800} sx={{ pb: 1 }}>
          <Box display="flex" alignItems="center" gap={1}>
            <GroupIcon color="primary" />
            Select Group Chat
          </Box>
          <Typography variant="caption" color="text.secondary" display="block" mt={0.5}>
            Choose a Soroush group from your account's chat list. The group ID will be auto-filled.
          </Typography>
        </DialogTitle>
        <DialogContent>
          {groupPickerLoading ? (
            <Box display="flex" flexDirection="column" alignItems="center" py={6} gap={2}>
              <CircularProgress size={36} />
              <Typography variant="body2" color="text.secondary">
                Connecting to Soroush and fetching groups...
              </Typography>
            </Box>
          ) : groupPickerError ? (
            <Alert severity="error" sx={{ borderRadius: 2 }}>
              {groupPickerError}
            </Alert>
          ) : (
            <>
              <TextField
                label="Search groups..."
                placeholder="Filter by name..."
                value={groupPickerSearch}
                onChange={(e) => setGroupPickerSearch(e.target.value)}
                fullWidth
                size="small"
                sx={{ mb: 2 }}
              />
              <Box sx={{
                maxHeight: 380,
                overflowY: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: 1,
                '&::-webkit-scrollbar': { width: '6px' },
                '&::-webkit-scrollbar-thumb': { bgcolor: 'rgba(255,255,255,0.1)', borderRadius: '3px' },
              }}>
                {groupList
                  .filter(g => g.title.toLowerCase().includes(groupPickerSearch.toLowerCase()))
                  .map((group) => (
                  <Paper
                    key={group.id}
                    variant="outlined"
                    onClick={() => handleSelectGroup(group)}
                    sx={{
                      p: 2,
                      cursor: 'pointer',
                      bgcolor: 'rgba(255,255,255,0.02)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: 2,
                      transition: 'all 0.2s ease',
                      '&:hover': {
                        bgcolor: 'rgba(139, 92, 246, 0.08)',
                        borderColor: 'rgba(139, 92, 246, 0.3)',
                        transform: 'translateX(4px)',
                      },
                    }}
                  >
                    <Box display="flex" alignItems="center" justifyContent="space-between">
                      <Box display="flex" alignItems="center" gap={1.5}>
                        <Box
                          sx={{
                            width: 36,
                            height: 36,
                            borderRadius: 2,
                            bgcolor: group.type === 'group' ? 'rgba(16, 185, 129, 0.15)' :
                                     group.type === 'supergroup' ? 'rgba(139, 92, 246, 0.15)' :
                                     'rgba(59, 130, 246, 0.15)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            flexShrink: 0,
                          }}
                        >
                          <GroupIcon sx={{
                            fontSize: 18,
                            color: group.type === 'group' ? '#10b981' :
                                   group.type === 'supergroup' ? '#a78bfa' : '#3b82f6'
                          }} />
                        </Box>
                        <Box>
                          <Typography variant="body2" fontWeight={600}>
                            {group.title}
                          </Typography>
                          <Box display="flex" alignItems="center" gap={1}>
                            <Chip
                              label={group.type}
                              size="small"
                              sx={{
                                height: 18,
                                fontSize: '0.65rem',
                                bgcolor: group.type === 'group' ? 'rgba(16,185,129,0.15)' : 'rgba(139,92,246,0.15)',
                                color: group.type === 'group' ? '#10b981' : '#a78bfa',
                              }}
                            />
                            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                              ID: {group.id}
                            </Typography>
                            {group.membersCount > 0 && (
                              <Typography variant="caption" color="text.secondary">
                                · {group.membersCount} members
                              </Typography>
                            )}
                          </Box>
                        </Box>
                      </Box>
                    </Box>
                  </Paper>
                ))}
                {groupList.filter(g => g.title.toLowerCase().includes(groupPickerSearch.toLowerCase())).length === 0 && (
                  <Box py={4} textAlign="center">
                    <Typography variant="body2" color="text.secondary">
                      {groupList.length === 0 ? 'No groups found in this account.' : 'No groups match your search.'}
                    </Typography>
                  </Box>
                )}
              </Box>
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2, pt: 0 }}>
          <Button customVariant="secondary" onClick={() => setGroupPickerOpen(false)}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 4: Real-time Engine Logs View (Polled from Go REST backend)
// ──────────────────────────────────────────────────────────────────────────────
function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [filter, setFilter] = useState<string>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const logContainerRef = useRef<HTMLDivElement>(null);

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

  const handleClearLogs = async () => {
    if (!confirm('Clear all logs? This cannot be undone.')) return;
    try {
      await fetch(`${API_BASE}/logs/clear`, { method: 'POST', headers: getHeaders() });
      fetchLogs();
    } catch { /* ignore */ }
  };

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 1500);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (autoScroll && logContainerRef.current) {
      logContainerRef.current.scrollTop = 0;
    }
  }, [logs, autoScroll]);

  const filteredLogs = filter === 'all' ? logs : logs.filter(l => l.type === filter);

  const logTypeCounts = {
    all: logs.length,
    info: logs.filter(l => l.type === 'info').length,
    success: logs.filter(l => l.type === 'success').length,
    warn: logs.filter(l => l.type === 'warn').length,
    error: logs.filter(l => l.type === 'error').length,
  };

  const typeColors: Record<string, string> = {
    info: '#94a3b8',
    success: '#10b981',
    warn: '#f59e0b',
    error: '#ef4444',
  };

  const filterChips: { key: string; label: string; color: string }[] = [
    { key: 'all', label: 'All', color: '#8b5cf6' },
    { key: 'info', label: 'Info', color: '#94a3b8' },
    { key: 'success', label: 'Success', color: '#10b981' },
    { key: 'warn', label: 'Warning', color: '#f59e0b' },
    { key: 'error', label: 'Error', color: '#ef4444' },
  ];

  return (
    <Card>
      <CardContent>
        {/* Header with controls */}
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
          <Box display="flex" alignItems="center" gap={1.5}>
            <Typography variant="h6" fontWeight={700}>
              Engine Logs
            </Typography>
            <Chip
              label={`${logs.length} entries`}
              size="small"
              sx={{
                bgcolor: 'rgba(139, 92, 246, 0.1)',
                color: '#a78bfa',
                fontWeight: 600,
                fontSize: '0.7rem',
              }}
            />
          </Box>
          <Box display="flex" alignItems="center" gap={1}>
            <Chip
              label={autoScroll ? "Auto-scroll ON" : "Auto-scroll OFF"}
              size="small"
              onClick={() => setAutoScroll(!autoScroll)}
              sx={{
                cursor: 'pointer',
                bgcolor: autoScroll ? 'rgba(16, 185, 129, 0.1)' : 'rgba(255,255,255,0.05)',
                color: autoScroll ? '#10b981' : '#64748b',
                fontWeight: 600,
                fontSize: '0.7rem',
                '&:hover': { bgcolor: autoScroll ? 'rgba(16, 185, 129, 0.2)' : 'rgba(255,255,255,0.1)' },
              }}
            />
            <IconButton onClick={fetchLogs} color="primary" size="small">
              <RefreshIcon fontSize="small" />
            </IconButton>
            <Button
              customVariant="danger"
              size="small"
              onClick={handleClearLogs}
              style={{ padding: '3px 10px', fontSize: '0.7rem', minWidth: 0 }}
            >
              Clear
            </Button>
          </Box>
        </Box>

        {/* Filter chips */}
        <Box display="flex" gap={1} mb={2} flexWrap="wrap">
          {filterChips.map(chip => (
            <Chip
              key={chip.key}
              label={`${chip.label} (${logTypeCounts[chip.key as keyof typeof logTypeCounts]})`}
              size="small"
              onClick={() => setFilter(chip.key)}
              sx={{
                cursor: 'pointer',
                bgcolor: filter === chip.key ? `${chip.color}22` : 'rgba(255,255,255,0.03)',
                color: filter === chip.key ? chip.color : '#64748b',
                border: '1px solid',
                borderColor: filter === chip.key ? `${chip.color}44` : 'transparent',
                fontWeight: filter === chip.key ? 700 : 500,
                fontSize: '0.72rem',
                transition: 'all 0.2s ease',
                '&:hover': { bgcolor: `${chip.color}15` },
              }}
            />
          ))}
        </Box>

        {/* Log container */}
        <Box
          ref={logContainerRef}
          sx={{
            height: 520,
            bgcolor: '#020617',
            borderRadius: 3,
            p: 2,
            overflowY: 'auto',
            fontFamily: '"JetBrains Mono", "Fira Code", "Courier New", monospace',
            fontSize: '0.8rem',
            display: 'flex',
            flexDirection: 'column',
            gap: 0.25,
            boxShadow: 'inset 0 2px 10px rgba(0,0,0,0.8)',
            '&::-webkit-scrollbar': { width: '6px' },
            '&::-webkit-scrollbar-track': { bgcolor: 'transparent' },
            '&::-webkit-scrollbar-thumb': { bgcolor: 'rgba(255,255,255,0.1)', borderRadius: '3px' },
          }}
        >
          {filteredLogs.length > 0 ? (
            filteredLogs.map((log, index) => (
              <Box
                key={index}
                display="flex"
                gap={1}
                sx={{
                  py: 0.4,
                  px: 1,
                  borderRadius: 1,
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.02)' },
                  transition: 'background-color 0.15s ease',
                }}
              >
                <Box
                  sx={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    bgcolor: typeColors[log.type] || '#94a3b8',
                    mt: '7px',
                    flexShrink: 0,
                    boxShadow: `0 0 6px ${typeColors[log.type] || '#94a3b8'}60`,
                  }}
                />
                <Box sx={{ color: '#475569', userSelect: 'none', flexShrink: 0, fontSize: '0.75rem', lineHeight: '1.6' }}>
                  {log.timestamp}
                </Box>
                <Box
                  sx={{
                    color: typeColors[log.type] || '#94a3b8',
                    lineHeight: '1.6',
                    wordBreak: 'break-word',
                  }}
                >
                  {log.message}
                </Box>
              </Box>
            ))
          ) : (
            <Box py={8} textAlign="center" color="text.secondary" display="flex" flexDirection="column" alignItems="center" gap={1}>
              <LogsIcon sx={{ fontSize: 40, opacity: 0.15 }} />
              <Typography variant="body2" color="text.secondary">
                {filter !== 'all' ? `No ${filter} logs found.` : 'No logs recorded yet. Activity will appear here in real-time.'}
              </Typography>
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
