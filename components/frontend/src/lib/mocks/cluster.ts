/**
 * Mock cluster information data
 */

export const mockClusterInfo = {
  isOpenShift: true,
  version: '4.14.0',
  consoleUrl: 'https://console-openshift-console.apps.demo.openshift.com',
  apiUrl: 'https://api.demo.openshift.com:6443',
  clusterName: 'demo-cluster',
  region: 'us-east-1',
  provider: 'AWS',
  features: {
    oauth: true,
    monitoring: true,
    logging: true,
    registry: true,
    gitops: true,
  },
};

export const mockVersion = {
  version: '1.0.0',
  commit: 'abc123def456',
  buildDate: '2025-11-01T10:00:00Z',
  goVersion: '1.21.3',
  platform: 'linux/amd64',
};


