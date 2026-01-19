import { render } from 'solid-js/web';
import { FileLibrary } from './components/FileLibrary';

const mountPoint = document.getElementById('file-library');

if (mountPoint) {
  render(() => <FileLibrary />, mountPoint);
}
