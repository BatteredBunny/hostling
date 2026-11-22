import "./LoadingScreen.css";

// The spinning mascot, the app's name and a loading bar over the bare page
// background. Inside the app it is paired with hiding the page itself (see
// `.app.loading` in LoadingScreen.css).
export function LoadingScreen(props: { mascot: string; title: string }) {
  return (
    <div
      class="loading-screen"
      role="status"
      aria-label="Loading page"
      aria-busy="true"
    >
      <img
        class="loading-mascot"
        src={props.mascot}
        alt=""
        width="128"
        height="115"
      />
      <p class="loading-title">{props.title}</p>
      <div class="loading-bar" />
    </div>
  );
}
