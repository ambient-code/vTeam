export type WizardData = {
  projectName: string;
  projectDisplayName: string;
  projectDescription: string;
  githubConnected: boolean;
  createdProjectName?: string;
};

export type WizardStep = {
  id: string;
  title: string;
  description: string;
  optional?: boolean;
};

