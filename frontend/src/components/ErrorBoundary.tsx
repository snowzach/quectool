import { Component, ReactNode } from "react";

export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) { return { error }; }
  componentDidCatch(error: Error) { console.error("ErrorBoundary:", error); }
  render() {
    if (this.state.error) {
      return (
        <div className="p-6 text-red-700">
          <h1 className="text-xl font-semibold mb-2">Something went wrong.</h1>
          <pre className="text-sm">{this.state.error.message}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}
