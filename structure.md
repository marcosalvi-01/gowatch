# issues with sidebar

1. oob swap
   - the sidebar state is being updated too late, always after the main request finishes, need to wait while the main request loads which is bad
   - either the fragment components need to know the sidebar state or we can add it in the handler but we still need to get it every time

2. server htmx event
   - worse than oob for the timeout since it has to wait for both the main request to finish, and then for the second request to the sidebar endpoint
   - better then oob for the code mantainability because we keep the sidebar code and the fragment/handler completely separate

3. JS
   - js sucks
   - there is still the issue with the sidebar info needed inside the page that is going to return the content for the /watched endpoint since it will need to render the sidebar with the correct info the first time

---

possible solution:
js + trigger on load the sidebar the first time.

doing this we have the layout have a div with the onload that triggers the sidebar with the refearer thing for the active (should work), then from then we will have the handling done using js
