import { TableSkeleton } from '@/components/skeletons';

export default function PermissionsLoading() {
  return <TableSkeleton rows={6} columns={4} />;
}
