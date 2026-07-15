import { Link } from "react-router-dom";
import { EmptyState, PageHeader } from "../components/ui";

export function NotFoundPage() {
  return <div className="page">
    <PageHeader eyebrow="Navigation" title="Page not found" description="The requested AreaFlow route does not exist." />
    <EmptyState message="This link is no longer available" action={<Link className="secondary-button" to="/">Return to Overview</Link>} />
  </div>;
}
