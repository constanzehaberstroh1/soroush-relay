import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#7c3aed', // Purple server primary
      light: '#a78bfa',
      dark: '#5b21b6',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#06b6d4', // Cyan cyber accent
      light: '#22d3ee',
      dark: '#0891b2',
    },
    background: {
      default: '#fdfdfd',
      paper: 'rgba(255, 255, 255, 0.75)',
    },
    text: {
      primary: '#0f172a',
      secondary: '#475569',
    },
    divider: 'rgba(15, 23, 42, 0.08)',
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
    h1: { fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontWeight: 700, letterSpacing: '-0.01em' },
    h3: { fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    button: { fontWeight: 600, textTransform: 'none' },
  },
  shape: {
    borderRadius: 16,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          background: 'rgba(255, 255, 255, 0.75) !important',
          backdropFilter: 'blur(16px) saturate(120%)',
          border: '1px solid rgba(15, 23, 42, 0.08)',
          borderRadius: 16,
          boxShadow: '0 8px 32px rgba(124, 58, 237, 0.04)',
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
      },
    },
  },
});

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#a78bfa', // Server violet
      light: '#c084fc',
      dark: '#6d28d9',
      contrastText: '#0f172a',
    },
    secondary: {
      main: '#22d3ee', // Cyber cyan
      light: '#67e8f9',
      dark: '#0891b2',
    },
    background: {
      default: '#03030d', // Deep night violet-black
      paper: 'rgba(13, 10, 28, 0.5)', // Translucent server card
    },
    text: {
      primary: '#f8fafc',
      secondary: '#94a3b8',
    },
    divider: 'rgba(255, 255, 255, 0.08)',
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
    h1: { fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontWeight: 700, letterSpacing: '-0.01em' },
    h3: { fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    button: { fontWeight: 600, textTransform: 'none' },
  },
  shape: {
    borderRadius: 16,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          background: 'rgba(13, 10, 28, 0.5) !important',
          backdropFilter: 'blur(16px) saturate(120%)',
          border: '1px solid rgba(255, 255, 255, 0.06)',
          borderRadius: 16,
          boxShadow: '0 10px 30px -10px rgba(0, 0, 0, 0.6)',
          transition: 'all 0.4s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            borderColor: 'rgba(124, 58, 237, 0.35)',
            boxShadow: '0 15px 35px -5px rgba(124, 58, 237, 0.05), 0 0 20px rgba(124, 58, 237, 0.05)',
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
      },
    },
  },
});
