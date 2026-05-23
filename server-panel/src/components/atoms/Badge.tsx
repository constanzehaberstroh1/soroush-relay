import React from 'react';
import { Box, Chip } from '@mui/material';
import type { ChipProps } from '@mui/material';

export interface AtomicBadgeProps extends Omit<ChipProps, 'color'> {
  customVariant?: 'online' | 'connecting' | 'offline' | 'idle';
}

const Dot = styled(Box)<{ color: string; pulsing?: boolean }>(({ color, pulsing }) => ({
  width: 8,
  height: 8,
  borderRadius: '50%',
  backgroundColor: color,
  marginRight: 6,
  display: 'inline-block',
  boxShadow: `0 0 8px ${color}`,
  ...(pulsing && {
    animation: 'pulse-dot 1.5s infinite ease-in-out',
  }),
  '@keyframes pulse-dot': {
    '0%': {
      transform: 'scale(0.8)',
      opacity: 0.5,
    },
    '50%': {
      transform: 'scale(1.2)',
      opacity: 1,
    },
    '100%': {
      transform: 'scale(0.8)',
      opacity: 0.5,
    },
  },
}));

import { styled } from '@mui/material/styles';

export const Badge: React.FC<AtomicBadgeProps> = ({ customVariant = 'offline', label, ...props }) => {
  let color = '#94a3b8'; // gray
  let bg = 'rgba(148, 163, 184, 0.1)';
  let text = '#64748b';
  let pulsing = false;

  switch (customVariant) {
    case 'online':
      color = '#10b981'; // green
      bg = 'rgba(16, 185, 129, 0.1)';
      text = '#059669';
      pulsing = true;
      break;
    case 'connecting':
      color = '#f59e0b'; // orange
      bg = 'rgba(245, 158, 11, 0.1)';
      text = '#d97706';
      pulsing = true;
      break;
    case 'idle':
      color = '#a78bfa'; // violet for server
      bg = 'rgba(167, 139, 250, 0.1)';
      text = '#7c3aed';
      break;
    case 'offline':
    default:
      color = '#ef4444'; // red
      bg = 'rgba(239, 68, 68, 0.1)';
      text = '#dc2626';
      break;
  }

  return (
    <Chip
      {...props}
      label={
        <Box display="flex" alignItems="center">
          <Dot color={color} pulsing={pulsing} />
          {label}
        </Box>
      }
      sx={{
        backgroundColor: bg,
        color: text,
        fontWeight: 700,
        fontSize: '0.75rem',
        borderRadius: '8px',
        border: `1px solid ${color}20`,
        fontFamily: '"Inter", sans-serif',
        height: 24,
        ...props.sx,
      }}
    />
  );
};

export default Badge;
