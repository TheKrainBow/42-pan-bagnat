export const tours = {
  'module-import': [
    {
      anchor: 'moduleImport.name',
      title: 'Module Name',
      desc: 'Admin-only label. You will be able to customize what users see later.',
    },
    {
      anchor: 'moduleImport.git',
      title: 'Git Repository URL',
      desc: 'Use HTTPS for public repos, SSH for private ones. SSH will trigger an SSH setup flow next.',
    },
    {
      anchor: 'moduleImport.branch',
      title: 'Git Branch Name',
      desc: 'Optionally choose a different branch to clone than the default (main).',
    },
    {
      anchor: 'moduleImport.submit',
      title: 'Import Module',
      desc: 'Click this button to import the module and continue the tutorial.',
      proceedOn: 'click',
      nextHidden: true,
    },
  ],
};
