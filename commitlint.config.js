export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat', // New feature
        'fix', // Bug fix
        'docs', // Documentation changes
        'style', // Code style changes (formatting, etc.)
        'refactor', // Code refactoring
        'perf', // Performance improvements
        'test', // Adding or updating tests
        'build', // Build system or dependency changes
        'ci', // CI/CD changes
        'chore', // Other changes that don't modify src or test files
        'revert', // Revert a previous commit
      ],
    ],
    'scope-enum': [
      2,
      'always',
      [
        'mobile', // Flutter mobile app
        'workers', // Workers service
        'api', // GraphQL API
        'database', // Database/Supabase
        'shared', // Shared packages
        'deps', // Dependencies
        'release', // Release-related
        'docker', // Docker configuration
        'ci', // CI/CD workflows
        'docs', // Documentation
      ],
    ],
    'subject-case': [2, 'never', ['upper-case']], // No UPPERCASE subjects
    'subject-empty': [2, 'never'], // Subject cannot be empty
    'subject-full-stop': [2, 'never', '.'], // No period at end
    'header-max-length': [2, 'always', 100], // Max 100 characters for header
    'body-leading-blank': [2, 'always'], // Blank line before body
    'footer-leading-blank': [2, 'always'], // Blank line before footer
  },
};
