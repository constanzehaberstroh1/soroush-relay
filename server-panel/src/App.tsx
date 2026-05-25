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
  List,
  ListItem,
  ListItemText,
  LinearProgress,
  Divider,
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
  Save as SaveIcon,
  PowerSettingsNew as ShutdownIcon,
  People as UsersIcon,
  Dns as ServerIcon,
  Speed as PerformanceIcon,
  Assessment as StatsIcon,
  Dashboard as DashboardIcon,
  HistoryToggleOff as LogsIcon,
  Settings as SettingsIcon,
  LockOutlined as LockIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  AccountCircle as AccountIcon,
  Refresh as RefreshIcon,
  NetworkCheck as NetworkCheckIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  Warning as WarningIcon,
  PlayArrow as PlayArrowIcon,
  Storage as StorageIcon,
  VpnKey as VpnKeyIcon,
  Wifi as WifiIcon,
  CloudDone as CloudDoneIcon,
} from '@mui/icons-material';
import { lightTheme, darkTheme } from './theme';
import {
  LineChart,
  Line,
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
interface ConnectedClient {
  id: string;
  ip: string;
  phone: string;
  uptime: string;
  bandwidth: string;
  status: 'active' | 'stale';
}

interface SoroushAccount {
  id: string;
  phoneNumber: string;
  name: string;
  soroushUserId: number;
  displayName: string;
  accessHash: number;
  dcId: number;
  role: string;
  status: 'connected' | 'error' | 'idle';
  lastActive: string;
  createdAt: string;
}

interface InfraTestResult {
  name: string;
  description: string;
  status: 'testing' | 'pass' | 'fail' | 'warn';
  latencyMs: number;
  detail: string;
  category: 'network' | 'auth' | 'turn' | 'database';
}

interface LogEntry {
  timestamp: string;
  type: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

// REST Backend Service Utility
const API_BASE = '/api';

const getHeaders = () => {
  const token = localStorage.getItem('server-admin-token');
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
    return localStorage.getItem('server-theme-mode') === 'dark';
  });

  const handleThemeToggle = () => {
    const nextMode = !darkMode;
    setDarkMode(nextMode);
    localStorage.setItem('server-theme-mode', nextMode ? 'dark' : 'light');
  };

  const handleLogout = () => {
    localStorage.removeItem('server-admin-token');
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
              <ServerIcon color="primary" sx={{ fontSize: 28 }} className="server-pulsing-glow" />
              <Typography variant="h6" color="text.primary" fontWeight={800} style={{ letterSpacing: '-0.01em' }}>
                SOROUSH RELAY
                <Box component="span" sx={{ color: 'primary.main', ml: 1, fontWeight: 400, fontSize: '0.85em' }}>
                  Server Exit Node Panel
                </Box>
              </Typography>
            </Box>
            
            <Box display="flex" alignItems="center" gap={2}>
              <Badge
                label="ENGINE: ACTIVE"
                customVariant="online"
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
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <DashboardIcon fontSize="small" /> System Health
            </Link>
            <Link
              to="/accounts"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <AccountIcon fontSize="small" /> Soroush exit Accounts
            </Link>
            <Link
              to="/config"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <SettingsIcon fontSize="small" /> Gateway Settings
            </Link>
            <Link
              to="/infrastructure"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <NetworkCheckIcon fontSize="small" /> Infrastructure
            </Link>
            <Link
              to="/logs"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <LogsIcon fontSize="small" /> Signaling Logs
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
// Page 1: Dashboard View (Live polled backend system metrics)
// ──────────────────────────────────────────────────────────────────────────────
function DashboardView() {
  const [cpuUsage, setCpuUsage] = useState<number>(12);
  const [memUsage, setMemUsage] = useState<number>(34);
  const [activeClientsCount, setActiveClientsCount] = useState<number>(0);
  const [connectedClients, setConnectedClients] = useState<ConnectedClient[]>([]);
  const [perfData, setPerfData] = useState<{ time: string; cpu: number; mem: number }[]>([]);

  const fetchStats = async () => {
    try {
      const res = await fetch(`${API_BASE}/stats`, { headers: getHeaders() });
      if (res.status === 401) {
        localStorage.removeItem('server-admin-token');
        window.location.reload();
        return;
      }
      if (res.ok) {
        const data = await res.json();
        setCpuUsage(data.cpu);
        setMemUsage(data.memory);
        setActiveClientsCount(data.activeTunnels);

        const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        setPerfData((prev) => [...prev, { time, cpu: data.cpu, mem: data.memory }].slice(-15));
      }
    } catch (err) {
      console.error('Failed fetching stats:', err);
    }
  };

  const fetchClients = async () => {
    try {
      const res = await fetch(`${API_BASE}/clients`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setConnectedClients(data);
      }
    } catch (err) {
      console.error('Failed fetching pipelines:', err);
    }
  };

  useEffect(() => {
    fetchStats();
    fetchClients();
    const interval = setInterval(() => {
      fetchStats();
      fetchClients();
    }, 2000);

    return () => clearInterval(interval);
  }, []);

  return (
    <Grid container spacing={3}>
      {/* Left Column: CPU & Connected Users */}
      <Grid item xs={12} md={4}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
                  <Typography variant="h6" fontWeight={700}>
                    Engine Utilization
                  </Typography>
                  <PerformanceIcon color="primary" />
                </Box>
                
                <Box display="flex" flexDirection="column" gap={2}>
                  <Box>
                    <Box display="flex" justifyContent="space-between" mb={0.5}>
                      <Typography variant="body2" color="text.secondary">CPU Load</Typography>
                      <Typography variant="body2" fontWeight="bold">{cpuUsage}%</Typography>
                    </Box>
                    <LinearProgress variant="determinate" value={cpuUsage} color={cpuUsage > 80 ? "error" : "primary"} />
                  </Box>

                  <Box>
                    <Box display="flex" justifyContent="space-between" mb={0.5}>
                      <Typography variant="body2" color="text.secondary">Memory Usage</Typography>
                      <Typography variant="body2" fontWeight="bold">{memUsage}%</Typography>
                    </Box>
                    <LinearProgress variant="determinate" value={memUsage} color={memUsage > 85 ? "error" : "secondary"} />
                  </Box>
                  
                  <Divider sx={{ my: 1 }} />
                  
                  <Grid container spacing={1}>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">Uptime</Typography>
                      <Typography variant="body2" fontWeight="bold">6d 18h 22m</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">Active Tunnels</Typography>
                      <Typography variant="body2" fontWeight="bold" color="primary.main">{activeClientsCount} Channels</Typography>
                    </Grid>
                  </Grid>
                </Box>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
                  <Typography variant="h6" fontWeight={700}>
                    Active Client Pipelines
                  </Typography>
                  <UsersIcon color="secondary" />
                </Box>
                <Typography variant="caption" color="text.secondary">
                  Decapsulating incoming WebRTC data streams.
                </Typography>

                <Divider sx={{ my: 2 }} />

                <List dense>
                  {connectedClients.length > 0 ? (
                    connectedClients.map((client) => (
                      <ListItem key={client.id} sx={{ px: 0 }}>
                        <ListItemText
                          primary={client.phone}
                          secondary={
                            <Box component="span" display="flex" flexDirection="column">
                              <span>Endpoint IP: {client.ip}</span>
                              <Box component="span" display="flex" gap={1.5} alignItems="center" mt={0.5}>
                                <Badge
                                  label={client.uptime}
                                  customVariant="online"
                                />
                                <Typography variant="caption" color="text.secondary">
                                  Rate: {client.bandwidth}
                                </Typography>
                              </Box>
                            </Box>
                          }
                        />
                      </ListItem>
                    ))
                  ) : (
                    <Box py={2} textAlign="center">
                      <Typography variant="body2" color="text.secondary">
                        No active client pipelines linked.
                      </Typography>
                    </Box>
                  )}
                </List>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Grid>

      {/* Right Column: Graphs */}
      <Grid item xs={12} md={8}>
        <Card>
          <CardContent sx={{ height: 460, display: 'flex', flexDirection: 'column' }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6" fontWeight={700}>
                System Utilization Timeline
              </Typography>
              <StatsIcon color="primary" />
            </Box>
            
            <Box sx={{ flexGrow: 1, minHeight: 0 }}>
              {perfData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={perfData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.06)" />
                    <XAxis dataKey="time" stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <YAxis domain={[0, 100]} stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <Tooltip contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155' }} />
                    <Line type="monotone" dataKey="cpu" stroke="#a78bfa" strokeWidth={2} dot={false} name="CPU (%)" />
                    <Line type="monotone" dataKey="mem" stroke="#22d3ee" strokeWidth={2} dot={false} name="Memory (%)" />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <Box display="flex" alignItems="center" justifyContent="center" height="100%">
                  <Typography variant="body2" color="text.secondary">
                    Awaiting core metrics timeline...
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
// Page 2: Soroush Exit Accounts Manager View
// ──────────────────────────────────────────────────────────────────────────────
function AccountsView() {
  const [accounts, setAccounts] = useState<SoroushAccount[]>([]);
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
      setErrorMsg('Error contacting Soroush signaling gateway.');
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
            Registered Soroush Exit Credentials
          </Typography>
          <Button customVariant="primary" startIcon={<AddIcon />} onClick={() => setOpenAddDialog(true)}>
            Add Account
          </Button>
        </Box>
        <Typography variant="body2" color="text.secondary" mb={3}>
          Link exit node messaging accounts to negotiate WebRTC signaling handshakes with inbound clients.
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
                  <TableRow key={acc.id} hover sx={{ '&:hover': { bgcolor: 'rgba(124, 58, 237, 0.04) !important' } }}>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={1.5}>
                        <Avatar sx={{ bgcolor: 'rgba(124, 58, 237, 0.1)', color: 'secondary.main', fontSize: '0.85rem', fontWeight: 'bold' }}>
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
              No registered Soroush exit credentials found. Link credentials to begin.
            </Typography>
          </Box>
        )}

        {/* Add Account Dialog Wizard */}
        <Dialog open={openAddDialog} onClose={() => { setOpenAddDialog(false); setErrorMsg(''); setStep(1); setHelperNotice(''); }} fullWidth maxWidth="xs">
          <DialogTitle fontWeight={800} style={{ paddingBottom: '8px' }}>
            Add Soroush Exit Account
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
                  placeholder="e.g. Server Exit Account"
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
// Page 3: Gateway Configuration View
// ──────────────────────────────────────────────────────────────────────────────
function ConfigView() {
  const [serverPort, setServerPort] = useState<number>(8080);
  const [socksHost, setSocksHost] = useState<string>('127.0.0.1');
  const [socksPort, setSocksPort] = useState<number>(1080);
  const [bandwidthLimit, setBandwidthLimit] = useState<number>(100);

  // Group Bus Configuration state
  const [groupChatId, setGroupChatId] = useState<string>('');
  const [groupAccessHash, setGroupAccessHash] = useState<number>(0);
  const [groupPsk, setGroupPsk] = useState<string>('');
  const [groupSaving, setGroupSaving] = useState(false);
  const [groupSaved, setGroupSaved] = useState(false);

  // Group Picker Modal state
  const [groupPickerOpen, setGroupPickerOpen] = useState(false);
  const [groupPickerLoading, setGroupPickerLoading] = useState(false);
  const [groupPickerError, setGroupPickerError] = useState('');
  const [groupPickerSearch, setGroupPickerSearch] = useState('');
  const [groupList, setGroupList] = useState<{ id: number; title: string; type: string; membersCount: number; accessHash: number }[]>([]);

  // Tunnel Engine state
  const [engineRunning, setEngineRunning] = useState(false);
  const [engineStarting, setEngineStarting] = useState(false);
  const [engineError, setEngineError] = useState('');

  const fetchConfig = async () => {
    try {
      const res = await fetch(`${API_BASE}/config`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setServerPort(data.serverPort);
        setSocksHost(data.socksHost);
        setSocksPort(data.socksPort);
        setBandwidthLimit(data.bandwidthLimit);
      }
    } catch (err) {
      console.error('Failed fetching server config:', err);
    }
  };

  // Load group config on mount
  useEffect(() => {
    fetchConfig();
    const loadGroupConfig = async () => {
      try {
        const res = await fetch(`${API_BASE}/group/config`, { headers: getHeaders() });
        if (res.ok) {
          const data = await res.json();
          if (data.groupChatId) setGroupChatId(String(data.groupChatId));
          if (data.groupAccessHash) setGroupAccessHash(data.groupAccessHash);
          if (data.psk) setGroupPsk(data.psk);
        }
      } catch { /* ignore */ }
    };
    loadGroupConfig();
    // Load tunnel engine status
    const loadEngineStatus = async () => {
      try {
        const res = await fetch(`${API_BASE}/tunnel/status`, { headers: getHeaders() });
        if (res.ok) {
          const data = await res.json();
          setEngineRunning(data.running || false);
        }
      } catch { /* ignore */ }
    };
    loadEngineStatus();
  }, []);

  const handleSaveConfig = async () => {
    try {
      const res = await fetch(`${API_BASE}/config`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          serverPort: Number(serverPort),
          socksHost,
          socksPort: Number(socksPort),
          bandwidthLimit: Number(bandwidthLimit),
        }),
      });
      if (res.ok) {
        alert('Server gateway settings updated successfully!');
      }
    } catch (err) {
      console.error('Failed saving server config:', err);
    }
  };

  const handleSaveGroupConfig = async () => {
    setGroupSaving(true);
    setGroupSaved(false);
    try {
      const res = await fetch(`${API_BASE}/group/config`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          groupChatId: Number(groupChatId),
          groupAccessHash: groupAccessHash,
          psk: groupPsk,
        }),
      });
      if (res.ok) setGroupSaved(true);
    } catch { /* ignore */ }
    finally { setGroupSaving(false); }
  };

  const handleRestartServer = () => {
    alert('Graceful restart trace dispatched to signaling coordinator.');
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

  const handleSelectGroup = (group: { id: number; title: string; accessHash?: number }) => {
    setGroupChatId(String(group.id));
    setGroupAccessHash(group.accessHash || 0);
    setGroupSaved(false);
    setGroupPickerOpen(false);
  };

  return (
    <Box display="flex" flexDirection="column" gap={3}>
      {/* Engine Gateway Settings */}
      <Card>
        <CardContent>
          <Typography variant="h6" fontWeight={700} mb={2}>
            Engine Gateway Settings
          </Typography>
          
          <Grid container spacing={3}>
            <Grid item xs={12} sm={6}>
              <TextField
                label="HTTP Gateway Listen Port"
                type="number"
                value={serverPort}
                onChange={(e) => setServerPort(Number(e.target.value))}
                fullWidth
                size="small"
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Rate Limit per WebRTC channel (MB/s)"
                type="number"
                value={bandwidthLimit}
                onChange={(e) => setBandwidthLimit(Number(e.target.value))}
                fullWidth
                size="small"
              />
            </Grid>
            <Grid item xs={12} sm={8}>
              <TextField
                label="Forward Target Host (SOCKS5/HTTP Proxy Target)"
                value={socksHost}
                onChange={(e) => setSocksHost(e.target.value)}
                fullWidth
                size="small"
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                label="Target Port"
                type="number"
                value={socksPort}
                onChange={(e) => setSocksPort(Number(e.target.value))}
                fullWidth
                size="small"
              />
            </Grid>

            <Grid item xs={12}>
              <Box display="flex" gap={2} mt={1}>
                <Button
                  customVariant="primary"
                  startIcon={<SaveIcon />}
                  onClick={handleSaveConfig}
                >
                  Apply Settings
                </Button>
                <Button
                  customVariant="danger"
                  startIcon={<ShutdownIcon />}
                  onClick={handleRestartServer}
                >
                  Restart Engine
                </Button>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Group Bus Configuration — The "Secret Rendezvous Point" */}
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" gap={1.5} mb={1}>
            <UsersIcon color="secondary" />
            <Typography variant="h6" fontWeight={700}>
              Group Bus Configuration
            </Typography>
          </Box>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Both the Client and Server must independently know the same Group Chat ID and Pre-Shared Key (PSK).
            This is the "secret rendezvous point" — configure the exact same values that are set on the client side.
          </Typography>

          <Divider sx={{ mb: 3 }} />

          {groupSaved && (
            <Alert severity="success" sx={{ mb: 2, borderRadius: 2 }}>
              Group bus configuration saved successfully! Ensure the client uses the same Group Chat ID and PSK.
            </Alert>
          )}

          <Grid container spacing={3}>
            <Grid item xs={12} sm={6}>
              <Box display="flex" gap={1} alignItems="flex-start">
                <TextField
                  label="Group Chat ID"
                  placeholder="e.g. -1001234567890"
                  value={groupChatId}
                  onChange={(e) => { setGroupChatId(e.target.value); setGroupSaved(false); }}
                  fullWidth
                  size="small"
                  helperText="The Soroush group chat ID where stealth commands are exchanged."
                  InputProps={{
                    style: { fontFamily: 'monospace' },
                  }}
                />
                <Button
                  customVariant="secondary"
                  size="small"
                  onClick={handleOpenGroupPicker}
                  style={{ minWidth: 0, padding: '6px 12px', whiteSpace: 'nowrap', marginTop: '1px' }}
                  startIcon={<UsersIcon style={{ fontSize: 16 }} />}
                >
                  Browse
                </Button>
              </Box>
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                label="Pre-Shared Key (PSK)"
                placeholder="e.g. my-secret-key-2026"
                value={groupPsk}
                onChange={(e) => { setGroupPsk(e.target.value); setGroupSaved(false); }}
                fullWidth
                size="small"
                helperText="Symmetric key used to encode/decode stealth commands in group chat messages."
                InputProps={{
                  style: { fontFamily: 'monospace' },
                }}
              />
            </Grid>

            <Grid item xs={12}>
              <Box display="flex" gap={2} mt={1}>
                <Button
                  customVariant="glow"
                  startIcon={groupSaving ? <CircularProgress size={18} color="inherit" /> : <SaveIcon />}
                  onClick={handleSaveGroupConfig}
                  disabled={groupSaving || !groupChatId}
                >
                  {groupSaving ? 'Saving...' : 'Save Group Config'}
                </Button>
              </Box>
            </Grid>
          </Grid>

          {/* Visual hint about matching configuration */}
          <Paper
            variant="outlined"
            sx={{
              mt: 3,
              p: 2,
              borderRadius: 2,
              bgcolor: 'rgba(124, 58, 237, 0.04)',
              borderColor: 'rgba(124, 58, 237, 0.15)',
            }}
          >
            <Typography variant="caption" color="text.secondary" display="block" mb={0.5} fontWeight={600}>
              ⚡ How It Works
            </Typography>
            <Typography variant="caption" color="text.secondary" display="block">
              Without a central server, both Client and Server discover each other by monitoring the same Soroush group chat.
              The client sends a stealth DISCOVER command (encoded with the PSK), and this server responds with an OFFER — 
              all hidden inside normal-looking group chat messages. Both sides must have identical Group Chat ID and PSK values.
            </Typography>
          </Paper>
        </CardContent>
      </Card>

      {/* Tunnel Engine Control */}
      <Card>
        <CardContent>
          <Box display="flex" alignItems="center" gap={1.5} mb={1}>
            <Box sx={{
              width: 10, height: 10, borderRadius: '50%',
              bgcolor: engineRunning ? '#10b981' : 'rgba(255,255,255,0.2)',
              boxShadow: engineRunning ? '0 0 8px rgba(16, 185, 129, 0.5)' : 'none',
              animation: engineRunning ? 'pulse 2s infinite' : 'none',
            }} />
            <Typography variant="h6" fontWeight={700}>
              Tunnel Engine
            </Typography>
            <Badge
              label={engineRunning ? 'RUNNING' : 'STOPPED'}
              customVariant={engineRunning ? 'online' : 'offline'}
            />
          </Box>
          <Typography variant="body2" color="text.secondary" mb={2}>
            The tunnel engine connects to Soroush and listens for client DISCOVER messages in the group chat.
            It must be running for clients to find and connect to this server.
          </Typography>

          {engineError && (
            <Alert severity="error" sx={{ mb: 2, borderRadius: 2 }}>
              {engineError}
            </Alert>
          )}

          <Box display="flex" gap={2}>
            <Button
              customVariant={engineRunning ? 'secondary' : 'primary'}
              startIcon={engineStarting ? <CircularProgress size={18} color="inherit" /> : (engineRunning ? <ShutdownIcon /> : <PlayArrowIcon />)}
              onClick={async () => {
                setEngineStarting(true);
                setEngineError('');
                try {
                  const endpoint = engineRunning ? 'tunnel/stop' : 'tunnel/start';
                  const res = await fetch(`${API_BASE}/${endpoint}`, {
                    method: 'POST',
                    headers: getHeaders(),
                  });
                  if (res.ok) {
                    setEngineRunning(!engineRunning);
                  } else {
                    const data = await res.json();
                    setEngineError(data.error || 'Failed to toggle engine.');
                  }
                } catch {
                  setEngineError('Connection error.');
                } finally {
                  setEngineStarting(false);
                }
              }}
              disabled={engineStarting}
              sx={{ minWidth: 180 }}
            >
              {engineStarting ? 'Working...' : engineRunning ? 'Stop Engine' : 'Start Engine'}
            </Button>
          </Box>
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
            border: '1px solid rgba(255,255,255,0.1)',
            borderRadius: 3,
            overflow: 'hidden',
          }
        }}
      >
        <DialogTitle sx={{ pb: 1, pt: 2.5, px: 3 }}>
          <Box display="flex" alignItems="center" gap={1.5}>
            <Box sx={{
              width: 40, height: 40, borderRadius: 2,
              bgcolor: 'rgba(139, 92, 246, 0.15)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              <UsersIcon sx={{ color: '#a78bfa', fontSize: 22 }} />
            </Box>
            <Box>
              <Typography variant="h6" fontWeight={700} sx={{ lineHeight: 1.3 }}>
                Select Group Chat
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
                Choose a group from your account's chat list
              </Typography>
            </Box>
          </Box>
        </DialogTitle>
        <DialogContent sx={{ px: 3, pb: 2 }}>
          {groupPickerLoading ? (
            <Box display="flex" flexDirection="column" alignItems="center" py={6} gap={2}>
              <CircularProgress size={40} />
              <Typography variant="body1" color="text.secondary">
                Connecting to Soroush and fetching groups...
              </Typography>
            </Box>
          ) : groupPickerError ? (
            <Alert severity="error" sx={{ borderRadius: 2, fontSize: '0.95rem' }}>
              {groupPickerError}
            </Alert>
          ) : (
            <>
              <TextField
                label="Search groups..."
                placeholder="Filter by name or ID..."
                value={groupPickerSearch}
                onChange={(e) => setGroupPickerSearch(e.target.value)}
                fullWidth
                size="small"
                sx={{ mb: 2 }}
              />
              <Box sx={{
                maxHeight: 400,
                overflowY: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: 1.5,
                '&::-webkit-scrollbar': { width: '6px' },
                '&::-webkit-scrollbar-thumb': { bgcolor: 'rgba(255,255,255,0.12)', borderRadius: '3px' },
              }}>
                {groupList
                  .filter(g => g.title.toLowerCase().includes(groupPickerSearch.toLowerCase()) ||
                               String(g.id).includes(groupPickerSearch))
                  .map((group) => (
                  <Paper
                    key={group.id}
                    variant="outlined"
                    onClick={() => handleSelectGroup(group)}
                    sx={{
                      p: 2,
                      cursor: 'pointer',
                      bgcolor: 'rgba(255,255,255,0.02)',
                      border: '1px solid rgba(255,255,255,0.08)',
                      borderRadius: 2.5,
                      transition: 'all 0.2s ease',
                      '&:hover': {
                        bgcolor: 'rgba(139, 92, 246, 0.1)',
                        borderColor: 'rgba(139, 92, 246, 0.4)',
                        transform: 'translateX(4px)',
                        boxShadow: '0 0 20px rgba(139, 92, 246, 0.08)',
                      },
                    }}
                  >
                    <Box display="flex" alignItems="center" gap={2}>
                      <Box
                        sx={{
                          width: 44,
                          height: 44,
                          borderRadius: 2.5,
                          bgcolor: group.type === 'group' ? 'rgba(16, 185, 129, 0.15)' :
                                   group.type === 'supergroup' ? 'rgba(139, 92, 246, 0.15)' :
                                   'rgba(59, 130, 246, 0.15)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          flexShrink: 0,
                        }}
                      >
                        <UsersIcon sx={{
                          fontSize: 22,
                          color: group.type === 'group' ? '#10b981' :
                                 group.type === 'supergroup' ? '#a78bfa' : '#3b82f6'
                        }} />
                      </Box>
                      <Box flex={1} minWidth={0}>
                        <Typography
                          variant="body1"
                          fontWeight={600}
                          sx={{
                            fontSize: '1rem',
                            lineHeight: 1.3,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {group.title}
                        </Typography>
                        <Box display="flex" alignItems="center" gap={1} mt={0.5}>
                          <Badge
                            label={group.type}
                            customVariant={group.type === 'group' ? 'online' : 'idle'}
                          />
                          <Typography variant="body2" color="text.secondary" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                            ID: {group.id}
                          </Typography>
                          {group.membersCount > 0 && (
                            <Typography variant="body2" color="text.secondary" sx={{ fontSize: '0.8rem' }}>
                              · {group.membersCount} members
                            </Typography>
                          )}
                        </Box>
                      </Box>
                      <Box sx={{
                        color: 'rgba(255,255,255,0.2)',
                        fontSize: '1.5rem',
                        transition: 'all 0.2s',
                        '.MuiPaper-root:hover &': { color: '#a78bfa', transform: 'translateX(2px)' },
                      }}>
                        ›
                      </Box>
                    </Box>
                  </Paper>
                ))}
                {groupList.filter(g => g.title.toLowerCase().includes(groupPickerSearch.toLowerCase())).length === 0 && (
                  <Box py={4} textAlign="center">
                    <Typography variant="body1" color="text.secondary">
                      {groupList.length === 0 ? 'No groups found in this account.' : 'No groups match your search.'}
                    </Typography>
                  </Box>
                )}
              </Box>
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2.5, pt: 1, borderTop: '1px solid rgba(255,255,255,0.06)' }}>
          <Button customVariant="secondary" onClick={() => setGroupPickerOpen(false)}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 5: Infrastructure Diagnostics View
// ──────────────────────────────────────────────────────────────────────────────

const categoryMeta: Record<string, { icon: React.ReactNode; label: string; color: string }> = {
  network: { icon: <WifiIcon />, label: 'Network Connectivity', color: '#22d3ee' },
  auth: { icon: <VpnKeyIcon />, label: 'Authentication & Accounts', color: '#a78bfa' },
  turn: { icon: <CloudDoneIcon />, label: 'TURN/STUN Relay Servers', color: '#f59e0b' },
  database: { icon: <StorageIcon />, label: 'Database Storage', color: '#10b981' },
};

function InfraTestView() {
  const [results, setResults] = useState<InfraTestResult[]>([]);
  const [testing, setTesting] = useState<boolean>(false);
  const [lastTested, setLastTested] = useState<string>('');
  const [cachedLoaded, setCachedLoaded] = useState<boolean>(false);

  // Load cached results on mount
  useEffect(() => {
    const loadCached = async () => {
      try {
        const res = await fetch(`${API_BASE}/infra/status`, { headers: getHeaders() });
        if (res.ok) {
          const data = await res.json();
          if (data.tested && data.results) {
            setResults(data.results);
            setLastTested(data.testedAt || '');
            setCachedLoaded(true);
          }
        }
      } catch { /* ignore */ }
    };
    loadCached();
  }, []);

  const runTests = async () => {
    setTesting(true);
    setResults([]);
    try {
      const res = await fetch(`${API_BASE}/infra/test`, {
        method: 'POST',
        headers: getHeaders(),
      });
      if (res.ok) {
        const data: InfraTestResult[] = await res.json();
        setResults(data);
        setLastTested(new Date().toISOString());
      }
    } catch (err) {
      console.error('Infrastructure test failed:', err);
    } finally {
      setTesting(false);
    }
  };

  const passCount = results.filter(r => r.status === 'pass').length;
  const failCount = results.filter(r => r.status === 'fail').length;
  const warnCount = results.filter(r => r.status === 'warn').length;
  const totalCount = results.length;

  const overallHealth = totalCount === 0
    ? 'unknown'
    : failCount > 0
    ? 'critical'
    : warnCount > 0
    ? 'degraded'
    : 'healthy';

  const healthColors: Record<string, string> = {
    unknown: '#475569',
    healthy: '#10b981',
    degraded: '#f59e0b',
    critical: '#ef4444',
  };

  // Group results by category
  const grouped = results.reduce<Record<string, InfraTestResult[]>>((acc, r) => {
    if (!acc[r.category]) acc[r.category] = [];
    acc[r.category].push(r);
    return acc;
  }, {});

  const StatusIcon = ({ status }: { status: string }) => {
    switch (status) {
      case 'pass':
        return <CheckCircleIcon sx={{ color: '#10b981', fontSize: 22 }} />;
      case 'fail':
        return <CancelIcon sx={{ color: '#ef4444', fontSize: 22 }} />;
      case 'warn':
        return <WarningIcon sx={{ color: '#f59e0b', fontSize: 22 }} />;
      default:
        return <CircularProgress size={18} />;
    }
  };

  return (
    <Box display="flex" flexDirection="column" gap={3}>
      {/* Header Card with overall health status */}
      <Card sx={{ overflow: 'visible' }}>
        <CardContent>
          <Box display="flex" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={2}>
            <Box display="flex" alignItems="center" gap={2}>
              <Box
                sx={{
                  width: 56,
                  height: 56,
                  borderRadius: 3,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: `linear-gradient(135deg, ${healthColors[overallHealth]}22, ${healthColors[overallHealth]}08)`,
                  border: `1px solid ${healthColors[overallHealth]}33`,
                  boxShadow: `0 0 20px ${healthColors[overallHealth]}15`,
                }}
              >
                <NetworkCheckIcon sx={{ color: healthColors[overallHealth], fontSize: 28 }} />
              </Box>
              <Box>
                <Typography variant="h6" fontWeight={800}>
                  Soroush Infrastructure Diagnostics
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Test all Soroush-related services and dependencies to verify server readiness
                </Typography>
              </Box>
            </Box>

            <Box display="flex" alignItems="center" gap={2}>
              {totalCount > 0 && (
                <Box display="flex" gap={1.5} alignItems="center">
                  <Box
                    sx={{
                      px: 1.5, py: 0.5, borderRadius: 2,
                      bgcolor: 'rgba(16, 185, 129, 0.1)',
                      border: '1px solid rgba(16, 185, 129, 0.2)',
                      display: 'flex', alignItems: 'center', gap: 0.5,
                    }}
                  >
                    <CheckCircleIcon sx={{ color: '#10b981', fontSize: 16 }} />
                    <Typography variant="caption" fontWeight={700} color="#10b981">{passCount}</Typography>
                  </Box>
                  {warnCount > 0 && (
                    <Box
                      sx={{
                        px: 1.5, py: 0.5, borderRadius: 2,
                        bgcolor: 'rgba(245, 158, 11, 0.1)',
                        border: '1px solid rgba(245, 158, 11, 0.2)',
                        display: 'flex', alignItems: 'center', gap: 0.5,
                      }}
                    >
                      <WarningIcon sx={{ color: '#f59e0b', fontSize: 16 }} />
                      <Typography variant="caption" fontWeight={700} color="#f59e0b">{warnCount}</Typography>
                    </Box>
                  )}
                  {failCount > 0 && (
                    <Box
                      sx={{
                        px: 1.5, py: 0.5, borderRadius: 2,
                        bgcolor: 'rgba(239, 68, 68, 0.1)',
                        border: '1px solid rgba(239, 68, 68, 0.2)',
                        display: 'flex', alignItems: 'center', gap: 0.5,
                      }}
                    >
                      <CancelIcon sx={{ color: '#ef4444', fontSize: 16 }} />
                      <Typography variant="caption" fontWeight={700} color="#ef4444">{failCount}</Typography>
                    </Box>
                  )}
                </Box>
              )}

              <Button
                customVariant="glow"
                startIcon={testing ? <CircularProgress size={18} color="inherit" /> : <PlayArrowIcon />}
                onClick={runTests}
                disabled={testing}
                style={{ minWidth: 160 }}
              >
                {testing ? 'Running Tests...' : 'Run All Tests'}
              </Button>
            </Box>
          </Box>

          {lastTested && (
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1.5, display: 'block' }}>
              Last tested: {new Date(lastTested).toLocaleString()}
            </Typography>
          )}
        </CardContent>
      </Card>

      {/* Overall health bar */}
      {totalCount > 0 && (
        <Box
          sx={{
            height: 6,
            borderRadius: 3,
            overflow: 'hidden',
            display: 'flex',
            bgcolor: 'rgba(255,255,255,0.04)',
          }}
        >
          <Box sx={{ width: `${(passCount / totalCount) * 100}%`, bgcolor: '#10b981', transition: 'width 0.5s' }} />
          <Box sx={{ width: `${(warnCount / totalCount) * 100}%`, bgcolor: '#f59e0b', transition: 'width 0.5s' }} />
          <Box sx={{ width: `${(failCount / totalCount) * 100}%`, bgcolor: '#ef4444', transition: 'width 0.5s' }} />
        </Box>
      )}

      {/* Testing skeleton */}
      {testing && results.length === 0 && (
        <Card>
          <CardContent>
            <Box display="flex" flexDirection="column" alignItems="center" py={6} gap={2}>
              <CircularProgress size={48} color="primary" />
              <Typography variant="body1" fontWeight={600}>
                Probing Soroush Infrastructure...
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Testing WebSocket gateway, TURN/STUN servers, MTProto handshake, database, and account health
              </Typography>
            </Box>
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {!testing && results.length === 0 && !cachedLoaded && (
        <Card>
          <CardContent>
            <Box display="flex" flexDirection="column" alignItems="center" py={8} gap={2}>
              <NetworkCheckIcon sx={{ fontSize: 64, opacity: 0.12, color: 'text.secondary' }} />
              <Typography variant="h6" fontWeight={600} color="text.secondary">
                No Test Results Yet
              </Typography>
              <Typography variant="body2" color="text.secondary" textAlign="center" maxWidth={440}>
                Click "Run All Tests" to probe all Soroush infrastructure components and verify this server has full access to the messaging network.
              </Typography>
            </Box>
          </CardContent>
        </Card>
      )}

      {/* Grouped results */}
      {Object.entries(grouped).map(([category, items]) => {
        const meta = categoryMeta[category] || { icon: <ServerIcon />, label: category, color: '#94a3b8' };
        const catPass = items.filter(i => i.status === 'pass').length;
        const catTotal = items.length;

        return (
          <Card key={category}>
            <CardContent>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
                <Box display="flex" alignItems="center" gap={1.5}>
                  <Box
                    sx={{
                      width: 36,
                      height: 36,
                      borderRadius: 2,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      bgcolor: `${meta.color}15`,
                      color: meta.color,
                    }}
                  >
                    {meta.icon}
                  </Box>
                  <Typography variant="subtitle1" fontWeight={700}>
                    {meta.label}
                  </Typography>
                </Box>
                <Typography variant="caption" color="text.secondary" fontWeight={600}>
                  {catPass}/{catTotal} passed
                </Typography>
              </Box>

              <Box display="flex" flexDirection="column" gap={1}>
                {items.map((item, idx) => (
                  <Box
                    key={idx}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 2,
                      px: 2,
                      py: 1.5,
                      borderRadius: 2,
                      bgcolor: item.status === 'pass'
                        ? 'rgba(16, 185, 129, 0.04)'
                        : item.status === 'fail'
                        ? 'rgba(239, 68, 68, 0.04)'
                        : item.status === 'warn'
                        ? 'rgba(245, 158, 11, 0.04)'
                        : 'transparent',
                      border: '1px solid',
                      borderColor: item.status === 'pass'
                        ? 'rgba(16, 185, 129, 0.12)'
                        : item.status === 'fail'
                        ? 'rgba(239, 68, 68, 0.12)'
                        : item.status === 'warn'
                        ? 'rgba(245, 158, 11, 0.12)'
                        : 'rgba(255,255,255,0.04)',
                      transition: 'all 0.3s ease',
                    }}
                  >
                    <StatusIcon status={item.status} />

                    <Box flex={1} minWidth={0}>
                      <Box display="flex" alignItems="center" gap={1}>
                        <Typography variant="body2" fontWeight={600} noWrap>
                          {item.name}
                        </Typography>
                        {item.latencyMs > 0 && (
                          <Typography
                            variant="caption"
                            sx={{
                              px: 1,
                              py: 0.25,
                              borderRadius: 1,
                              bgcolor: 'rgba(255,255,255,0.05)',
                              fontFamily: 'monospace',
                              fontWeight: 600,
                              color: item.latencyMs < 100 ? '#10b981'
                                : item.latencyMs < 500 ? '#f59e0b'
                                : '#ef4444',
                            }}
                          >
                            {item.latencyMs}ms
                          </Typography>
                        )}
                      </Box>
                      <Typography variant="caption" color="text.secondary" noWrap>
                        {item.detail || item.description}
                      </Typography>
                    </Box>

                    <Badge
                      label={item.status.toUpperCase()}
                      customVariant={
                        item.status === 'pass' ? 'online'
                        : item.status === 'fail' ? 'offline'
                        : item.status === 'warn' ? 'connecting'
                        : 'idle'
                      }
                    />
                  </Box>
                ))}
              </Box>
            </CardContent>
          </Card>
        );
      })}
    </Box>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 6: Logs View (Live polled backend signaling logs)
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
      console.error('Failed fetching signaling logs:', err);
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
    { key: 'all', label: 'All', color: '#a78bfa' },
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
              Signaling & P2P Routing Logs
            </Typography>
            <Badge
              label={`${logs.length} entries`}
              customVariant="idle"
            />
          </Box>
          <Box display="flex" alignItems="center" gap={1}>
            <Badge
              label={autoScroll ? "Auto-scroll ON" : "Auto-scroll OFF"}
              customVariant={autoScroll ? "online" : "idle"}
            />
            <IconButton onClick={() => setAutoScroll(!autoScroll)} color="inherit" size="small">
              <LogsIcon fontSize="small" />
            </IconButton>
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
            <Box
              key={chip.key}
              onClick={() => setFilter(chip.key)}
              sx={{
                cursor: 'pointer',
                px: 1.5,
                py: 0.5,
                borderRadius: 2,
                fontSize: '0.72rem',
                fontWeight: filter === chip.key ? 700 : 500,
                bgcolor: filter === chip.key ? `${chip.color}22` : 'rgba(255,255,255,0.03)',
                color: filter === chip.key ? chip.color : '#64748b',
                border: '1px solid',
                borderColor: filter === chip.key ? `${chip.color}44` : 'transparent',
                transition: 'all 0.2s ease',
                '&:hover': { bgcolor: `${chip.color}15` },
                userSelect: 'none',
              }}
            >
              {chip.label} ({logTypeCounts[chip.key as keyof typeof logTypeCounts]})
            </Box>
          ))}
        </Box>

        {/* Log container */}
        <Box
          ref={logContainerRef}
          sx={{
            height: 520,
            bgcolor: '#050212',
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
        localStorage.setItem('server-admin-token', data.token);
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
            <Avatar sx={{ m: 1, bgcolor: 'primary.main', width: 56, height: 56, boxShadow: '0 0 20px rgba(139, 92, 246, 0.4)' }}>
              <LockIcon style={{ color: '#0f172a', fontSize: 28 }} />
            </Avatar>
            <Typography variant="h5" component="h1" fontWeight={800} mt={1} color="text.primary" textAlign="center">
              SOROUSH RELAY
            </Typography>
            <Typography variant="caption" color="text.secondary" textAlign="center">
              Server Administrator Dashboard Portal
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

const configRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/config',
  component: ConfigView,
});

const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/logs',
  component: LogsView,
});

const infraRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/infrastructure',
  component: InfraTestView,
});

const routeTree = rootRoute.addChildren([indexRoute, accountsRoute, configRoute, infraRoute, logsRoute]);

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
    return localStorage.getItem('server-admin-token');
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
          localStorage.removeItem('server-admin-token');
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
