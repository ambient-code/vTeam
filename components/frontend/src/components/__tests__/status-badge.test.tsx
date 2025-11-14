import { render, screen } from '@testing-library/react';
import { StatusBadge, SessionPhaseBadge, ProjectStatusBadge, RFEPhaseBadge } from '../status-badge';

describe('StatusBadge', () => {
  it('renders with default status', () => {
    render(<StatusBadge status="default" />);
    expect(screen.getByText('Unknown')).toBeInTheDocument();
  });

  it('renders with success status', () => {
    render(<StatusBadge status="success" />);
    expect(screen.getByText('Success')).toBeInTheDocument();
  });

  it('renders with custom label', () => {
    render(<StatusBadge status="success" label="Custom Label" />);
    expect(screen.getByText('Custom Label')).toBeInTheDocument();
  });

  it('renders without icon when showIcon is false', () => {
    const { container } = render(<StatusBadge status="success" showIcon={false} />);
    const svg = container.querySelector('svg');
    expect(svg).not.toBeInTheDocument();
  });

  it('renders with icon by default', () => {
    const { container } = render(<StatusBadge status="success" />);
    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('normalizes status to lowercase', () => {
    render(<StatusBadge status="SUCCESS" label="Success" />);
    expect(screen.getByText('Success')).toBeInTheDocument();
  });
});

describe('SessionPhaseBadge', () => {
  it('renders pending phase', () => {
    render(<SessionPhaseBadge phase="pending" />);
    expect(screen.getByText('pending')).toBeInTheDocument();
  });

  it('renders running phase', () => {
    render(<SessionPhaseBadge phase="running" />);
    expect(screen.getByText('running')).toBeInTheDocument();
  });

  it('renders completed phase', () => {
    render(<SessionPhaseBadge phase="completed" />);
    expect(screen.getByText('completed')).toBeInTheDocument();
  });

  it('renders failed phase', () => {
    render(<SessionPhaseBadge phase="failed" />);
    expect(screen.getByText('failed')).toBeInTheDocument();
  });
});

describe('ProjectStatusBadge', () => {
  it('renders active status', () => {
    render(<ProjectStatusBadge status="active" />);
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('renders archived status', () => {
    render(<ProjectStatusBadge status="archived" />);
    expect(screen.getByText('archived')).toBeInTheDocument();
  });
});

describe('RFEPhaseBadge', () => {
  it('renders pre phase', () => {
    render(<RFEPhaseBadge phase="pre" />);
    expect(screen.getByText('pre')).toBeInTheDocument();
  });

  it('renders completed phase', () => {
    render(<RFEPhaseBadge phase="completed" />);
    expect(screen.getByText('completed')).toBeInTheDocument();
  });
});

