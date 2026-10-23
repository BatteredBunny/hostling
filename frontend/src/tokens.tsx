import { render } from 'solid-js/web';
import { TokensPage } from './components/TokensPage';

const mountPoint = document.getElementById('tokens-page');
if (mountPoint) render(() => <TokensPage />, mountPoint);
