import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  quickstartSidebar: [
    {
      type: 'category',
      label: '快速入门',
      items: ['intro'],
    },
  ],
  productDocsSidebar: [
    {
      type: 'category',
      label: '系统管理',
      items: [
        'system-management/user-management',
        'system-management/role-management',
        'system-management/menu-management',
        'system-management/session-management',
        'system-management/system-settings',
        'system-management/task-management',
        'system-management/announcement-management',
        'system-management/channel-management',
        'system-management/auth-provider-management',
        'system-management/identity-source-management',
        'system-management/audit-log-management',
      ],
    },
    {
      type: 'category',
      label: '许可管理',
      items: [
        'license-management/product-management',
        'license-management/licensee-management',
        'license-management/license-issuance',
      ],
    },
    {
      type: 'category',
      label: 'ITSM',
      items: [
        'itsm-management/service-catalog-management',
        'itsm-management/ticket-monitoring',
        'itsm-management/engine-settings',
      ],
    },
  ],
};

export default sidebars;
