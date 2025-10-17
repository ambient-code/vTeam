import { TableSkeleton } from '@/components/skeletons';

export default function SessionsLoading() {
  return <TableSkeleton rows={10} columns={5} />;
}
