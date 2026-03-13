<script lang="ts">
  import UserList from '@ui/components/admin/UserList/UserList.svelte';
</script>

<UserList
  items={[
    {
      id: 1,
      title: 'Jordan Lee',
      subtitle: 'Operations lead',
      cells: [
        {
          id: 'team',
          badge: {
            label: 'Admin',
            variant: 'neutral',
          },
        },
        {
          id: 'lifecycle',
          badge: {
            label: 'Active',
            variant: 'success',
          },
        },
        {
          id: 'action',
          align: 'end',
          content: 'Manage account',
        },
      ],
    },
    {
      id: 2,
      title: 'Sam Park',
      subtitle: 'Pending onboarding confirmation',
      cells: [
        {
          id: 'team',
          badge: {
            label: 'Editor',
            variant: 'neutral',
          },
        },
        {
          id: 'lifecycle',
          badge: {
            label: 'Invited',
            variant: 'warning',
          },
        },
        {
          id: 'action',
          align: 'end',
          content: 'Resend invite',
        },
      ],
    },
  ]}
/>
