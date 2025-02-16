import { extendTheme } from '@chakra-ui/react';
import colors from './colors';
import fontSizes from './fontSizes';
import breakpoints from './breakpoints';
import Button from './components/Button';
import Input from './components/Input';
import Select from './components/Select';
import Heading from './components/Heading';
import { mode, StyleFunctionProps } from '@chakra-ui/theme-tools';

const config = {
  cssVarPrefix: 'ck',
  initialColorMode: 'light',
  useSystemColorMode: false
};

const theme = extendTheme({
  colors,
  fontSizes,
  breakpoints,
  config,
  components: {
    Button,
    Input,
    Select,
    Heading
  },
  styles: {
    global: (props: StyleFunctionProps) => ({
      h1: mode('red', 'green')(props)
    })
  }
});

export default theme;
