/**
 * Creates and returns a result container for displaying verification results
 * @returns {HTMLElement} The result container
 */
function createResultContainer() {
    const container = document.createElement('div');
    container.className = 'verification-result';
    container.style.cssText = `
        font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
        background-color: #121212;
        border-radius: 16px;
        color: #ffffff;
        overflow: hidden;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
        margin: 32px auto;
        max-width: 960px;
    `;
    return container;
}

/**
 * Creates the header for the verification container
 * @param {string} title - The title to display
 * @returns {HTMLElement} The header element
 */
function createVerificationHeader(title = "Transaction Verification") {
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 24px;
        background: linear-gradient(135deg, #2c2c2c 0%, #1a1a1a 100%);
        border-bottom: 1px solid #333;
    `;
    
    const titleEl = document.createElement('h1');
    titleEl.textContent = title;
    titleEl.style.cssText = `
        color: #fff;
        font-size: 24px;
        font-weight: 700;
        margin: 0;
    `;
    
    header.appendChild(titleEl);
    return header;
}

/**
 * Creates a tab navigation system
 * @param {string[]} tabNames - Names of the tabs
 * @returns {Object} Object containing the tab container and content elements
 */
function createTabNavigation(tabNames) {
    const tabContainer = document.createElement('div');
    tabContainer.style.cssText = `
        display: flex;
        background: #1a1a1a;
        position: sticky;
        top: 0;
        z-index: 10;
    `;
    
    const tabContents = document.createElement('div');
    tabContents.style.cssText = `padding: 0;`;
    
    const contentSections = [];
    
    tabNames.forEach((name, index) => {
        // Create tab button
        const tab = document.createElement('button');
        tab.textContent = name;
        tab.style.cssText = `
            flex: 1;
            background: ${index === 0 ? '#272727' : 'transparent'};
            color: ${index === 0 ? '#03dac6' : '#888'};
            border: none;
            padding: 16px;
            font-size: 15px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            border-bottom: ${index === 0 ? '3px solid #03dac6' : '3px solid transparent'};
        `;
        tabContainer.appendChild(tab);
        
        // Create content section
        const content = document.createElement('div');
        content.style.cssText = `
            display: ${index === 0 ? 'block' : 'none'};
            padding: 24px;
        `;
        tabContents.appendChild(content);
        contentSections.push(content);
        
        // Add event listener
        tab.addEventListener('click', () => {
            // Update tab styles
            Array.from(tabContainer.children).forEach((t, i) => {
                t.style.background = i === index ? '#272727' : 'transparent';
                t.style.color = i === index ? '#03dac6' : '#888';
                t.style.borderBottom = i === index ? '3px solid #03dac6' : '3px solid transparent';
            });
            
            // Show selected content
            contentSections.forEach((c, i) => {
                c.style.display = i === index ? 'block' : 'none';
            });
        });
    });
    
    return { tabContainer, contentSections };
}

/**
 * Creates a verification status banner
 * @param {boolean} isNested - Whether this is a nested transaction
 * @returns {HTMLElement} The status banner
 */
function createStatusBanner(isNested = false) {
    const banner = document.createElement('div');
    
    if (isNested) {
        banner.style.cssText = `
            background: linear-gradient(90deg, rgba(255,82,82,0.15) 0%, rgba(255,82,82,0.05) 100%);
            border-left: 4px solid #ff5252;
            padding: 16px 24px;
            margin: 0 0 24px 0;
        `;
        
        const icon = document.createElement('span');
        icon.textContent = '⚠️';
        icon.style.cssText = 'font-size: 20px; margin-right: 12px; vertical-align: middle;';
        
        const text = document.createElement('span');
        text.innerHTML = '<strong>Nested Transaction Detected</strong> - This transaction approves another transaction. Verify both carefully.';
        text.style.cssText = 'vertical-align: middle; font-size: 15px;';
        
        banner.appendChild(icon);
        banner.appendChild(text);
    } else {
        banner.style.cssText = `
            background: linear-gradient(90deg, rgba(3,218,198,0.15) 0%, rgba(3,218,198,0.05) 100%);
            border-left: 4px solid #03dac6;
            padding: 16px 24px;
            margin: 0 0 24px 0;
        `;
        
        const icon = document.createElement('span');
        icon.textContent = '🔍';
        icon.style.cssText = 'font-size: 20px; margin-right: 12px; vertical-align: middle;';
        
        const text = document.createElement('span');
        text.innerHTML = '<strong>Verify Transaction Details</strong> - Ensure all details match what you expect.';
        text.style.cssText = 'vertical-align: middle; font-size: 15px;';
        
        banner.appendChild(icon);
        banner.appendChild(text);
    }
    
    return banner;
}

/**
 * Creates an info card for displaying grouped information
 * @param {string} title - Card title
 * @param {Object} data - Key-value pairs to display
 * @param {Object} options - Display options
 * @returns {HTMLElement} The info card
 */
function createInfoCard(title, data, options = {}) {
    const card = document.createElement('div');
    card.style.cssText = `
        background: #1e1e1e;
        border-radius: 12px;
        overflow: hidden;
        margin-bottom: 24px;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    `;
    
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #272727;
        border-bottom: 1px solid #333;
        display: flex;
        align-items: center;
        justify-content: space-between;
    `;
    
    const titleEl = document.createElement('h3');
    titleEl.textContent = title;
    titleEl.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #fff;
    `;
    
    header.appendChild(titleEl);
    
    // Add collapse button if collapsible
    if (options.collapsible) {
        const collapseBtn = document.createElement('button');
        collapseBtn.innerHTML = '−';
        collapseBtn.style.cssText = `
            background: none;
            border: none;
            color: #888;
            font-size: 20px;
            cursor: pointer;
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 0;
        `;
        
        header.appendChild(collapseBtn);
        
        // Create content container
        const content = document.createElement('div');
        content.style.cssText = `padding: 20px;`;
        
        // Toggle collapse on click
        collapseBtn.addEventListener('click', () => {
            if (content.style.display === 'none') {
                content.style.display = 'block';
                collapseBtn.innerHTML = '−';
            } else {
                content.style.display = 'none';
                collapseBtn.innerHTML = '+';
            }
        });
        
        card.appendChild(header);
        card.appendChild(content);
        
        // Add items to the card
        populateCardContent(content, data, options);
        
        return card;
    } else {
        card.appendChild(header);
        
        // Create content container
        const content = document.createElement('div');
        content.style.cssText = `padding: 20px;`;
        card.appendChild(content);
        
        // Add items to the card
        populateCardContent(content, data, options);
        
        return card;
    }
}

/**
 * Populates a card with content based on data
 * @param {HTMLElement} container - Container to populate
 * @param {Object} data - Data to display
 * @param {Object} options - Display options
 */
function populateCardContent(container, data, options) {
    if (options.type === 'hash') {
        // Display hashes with copy buttons
        Object.entries(data).forEach(([key, value]) => {
            const item = document.createElement('div');
            item.style.cssText = 'margin-bottom: 16px; last-child: margin-bottom: 0;';
            
            const label = document.createElement('div');
            label.textContent = key.charAt(0).toUpperCase() + key.slice(1).replace(/([A-Z])/g, ' $1');
            label.style.cssText = 'font-size: 14px; color: #888; margin-bottom: 6px;';
            
            const hashContainer = document.createElement('div');
            hashContainer.style.cssText = `
                display: flex;
                align-items: center;
                background: rgba(0,0,0,0.2);
                border-radius: 8px;
                padding: 10px 12px;
                position: relative;
            `;
            
            const hashValue = document.createElement('code');
            hashValue.textContent = value;
            hashValue.style.cssText = `
                font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
                color: #03dac6;
                overflow-x: auto;
                white-space: nowrap;
                font-size: 14px;
                margin-right: 10px;
                flex: 1;
            `;
            
            const copyBtn = document.createElement('button');
            copyBtn.textContent = 'Copy';
            copyBtn.style.cssText = `
                background: #333;
                border: none;
                color: #fff;
                padding: 6px 12px;
                border-radius: 4px;
                cursor: pointer;
                font-size: 12px;
                transition: background 0.2s;
                flex-shrink: 0;
            `;
            
            copyBtn.addEventListener('mouseover', () => {
                copyBtn.style.background = '#444';
            });
            
            copyBtn.addEventListener('mouseout', () => {
                copyBtn.style.background = '#333';
            });
            
            copyBtn.addEventListener('click', () => {
                navigator.clipboard.writeText(value);
                copyBtn.textContent = 'Copied!';
                setTimeout(() => { copyBtn.textContent = 'Copy'; }, 2000);
            });
            
            hashContainer.appendChild(hashValue);
            hashContainer.appendChild(copyBtn);
            
            item.appendChild(label);
            item.appendChild(hashContainer);
            container.appendChild(item);
        });
    } else if (options.type === 'table') {
        // Create table for key-value pairs
        const table = document.createElement('div');
        table.style.cssText = `
            border-radius: 8px;
            overflow: hidden;
            font-size: 14px;
        `;
        
        Object.entries(data).forEach(([key, value], index, array) => {
            const row = document.createElement('div');
            row.style.cssText = `
                display: flex;
                background: ${index % 2 === 0 ? 'rgba(0,0,0,0.1)' : 'rgba(0,0,0,0.2)'};
                ${index === array.length - 1 ? '' : 'border-bottom: 1px solid rgba(255,255,255,0.05);'}
            `;
            
            const keyCell = document.createElement('div');
            keyCell.textContent = key.charAt(0).toUpperCase() + key.slice(1).replace(/([A-Z])/g, ' $1');
            keyCell.style.cssText = `
                width: 140px;
                padding: 10px 16px;
                font-weight: 500;
                color: #bb86fc;
                flex-shrink: 0;
            `;
            
            const valueCell = document.createElement('div');
            valueCell.style.cssText = `
                flex: 1;
                padding: 10px 16px;
                word-break: break-word;
            `;
            
            // Detect verified addresses and highlight them
            if (typeof value === 'string' && value.includes(' ✅')) {
                const parts = value.split(' (');
                const address = document.createElement('span');
                address.textContent = parts[0];
                
                valueCell.appendChild(address);
                
                if (parts.length > 1) {
                    const verified = document.createElement('span');
                    verified.innerHTML = ` (<span style="color: #03dac6; font-weight: 500;">${parts[1]}</span>`;
                    valueCell.appendChild(verified);
                }
            } else {
                valueCell.textContent = value;
            }
            
            row.appendChild(keyCell);
            row.appendChild(valueCell);
            table.appendChild(row);
        });
        
        container.appendChild(table);
    } else {
        // Default display for key-value pairs
        Object.entries(data).forEach(([key, value]) => {
            const item = document.createElement('div');
            item.style.cssText = 'margin-bottom: 12px; display: flex; flex-wrap: wrap;';
            
            const label = document.createElement('div');
            label.textContent = key.charAt(0).toUpperCase() + key.slice(1).replace(/([A-Z])/g, ' $1');
            label.style.cssText = 'width: 140px; font-weight: 500; color: #bb86fc; margin-right: 16px; flex-shrink: 0;';
            
            const valueEl = document.createElement('div');
            valueEl.style.cssText = 'flex: 1; min-width: 200px;';
            
            // Detect verified addresses and highlight them
            if (typeof value === 'string' && value.includes(' ✅')) {
                const parts = value.split(' (');
                const address = document.createElement('span');
                address.textContent = parts[0];
                
                valueEl.appendChild(address);
                
                if (parts.length > 1) {
                    const verified = document.createElement('span');
                    verified.innerHTML = ` (<span style="color: #03dac6; font-weight: 500;">${parts[1]}</span>`;
                    valueEl.appendChild(verified);
                }
            } else {
                valueEl.textContent = value;
            }
            
            item.appendChild(label);
            item.appendChild(valueEl);
            container.appendChild(item);
        });
    }
}

/**
 * Creates a function call display
 * @param {Object} call - Call data
 * @returns {HTMLElement} The function call element
 */
function createFunctionCall(call) {
    const callContainer = document.createElement('div');
    callContainer.style.cssText = `
        background: #1e1e1e;
        border-radius: 12px;
        overflow: hidden;
        margin-bottom: 24px;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    `;
    
    // Format the target with contract name if known
    let targetDisplay = call.target;
    if (call.targetName) {
        targetDisplay = `${call.target} (${call.targetName} ✅)`;
    }
    
    // Create function header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #272727;
        display: flex;
        align-items: center;
        justify-content: space-between;
        border-bottom: 1px solid #333;
    `;
    
    const functionName = document.createElement('div');
    functionName.style.cssText = `
        font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
        font-weight: 600;
        font-size: 16px;
        color: #03dac6;
    `;
    functionName.textContent = call.functionName;
    
    header.appendChild(functionName);
    
    // Create collapse button
    const collapseBtn = document.createElement('button');
    collapseBtn.innerHTML = '−';
    collapseBtn.style.cssText = `
        background: none;
        border: none;
        color: #888;
        font-size: 20px;
        cursor: pointer;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0;
    `;
    
    header.appendChild(collapseBtn);
    callContainer.appendChild(header);
    
    // Create content container
    const content = document.createElement('div');
    content.style.cssText = `padding: 0;`;
    callContainer.appendChild(content);
    
    // Toggle collapse on click
    collapseBtn.addEventListener('click', () => {
        if (content.style.display === 'none') {
            content.style.display = 'block';
            collapseBtn.innerHTML = '−';
        } else {
            content.style.display = 'none';
            collapseBtn.innerHTML = '+';
        }
    });
    
    // Target address
    const targetSection = document.createElement('div');
    targetSection.style.cssText = `
        padding: 16px 20px;
        border-bottom: 1px solid rgba(255,255,255,0.05);
    `;
    
    const targetLabel = document.createElement('div');
    targetLabel.textContent = 'Contract Address';
    targetLabel.style.cssText = 'font-size: 14px; color: #888; margin-bottom: 6px;';
    
    const targetValue = document.createElement('div');
    targetValue.style.cssText = `
        font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
        word-break: break-all;
        background: rgba(0,0,0,0.2);
        padding: 10px;
        border-radius: 6px;
        font-size: 14px;
    `;
    
    // Detect verified addresses and highlight them
    if (targetDisplay.includes(' ✅')) {
        const parts = targetDisplay.split(' (');
        const address = document.createElement('span');
        address.textContent = parts[0];
        
        targetValue.appendChild(address);
        
        if (parts.length > 1) {
            const verified = document.createElement('span');
            verified.innerHTML = ` (<span style="color: #03dac6; font-weight: 500;">${parts[1]}</span>`;
            targetValue.appendChild(verified);
        }
    } else {
        targetValue.textContent = targetDisplay;
    }
    
    targetSection.appendChild(targetLabel);
    targetSection.appendChild(targetValue);
    content.appendChild(targetSection);
    
    // Parameters section if available
    if (call.parsedData) {
        const paramsSection = document.createElement('div');
        paramsSection.style.cssText = `
            padding: 16px 20px;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        `;
        
        const paramsLabel = document.createElement('div');
        paramsLabel.textContent = 'Parameters';
        paramsLabel.style.cssText = 'font-size: 14px; color: #888; margin-bottom: 10px;';
        
        paramsSection.appendChild(paramsLabel);
        
        // Create parameters table
        if (typeof call.parsedData === 'object') {
            const paramsTable = document.createElement('div');
            paramsTable.style.cssText = `
                border-radius: 6px;
                overflow: hidden;
                font-size: 14px;
            `;
            
            const keys = Object.keys(call.parsedData).sort();
            for (let i = 0; i < keys.length; i++) {
                const key = keys[i];
                const value = call.parsedData[key];
                
                const row = document.createElement('div');
                row.style.cssText = `
                    display: flex;
                    background: ${i % 2 === 0 ? 'rgba(0,0,0,0.1)' : 'rgba(0,0,0,0.2)'};
                    ${i === keys.length - 1 ? '' : 'border-bottom: 1px solid rgba(255,255,255,0.05);'}
                `;
                
                const keyCell = document.createElement('div');
                keyCell.textContent = key;
                keyCell.style.cssText = `
                    width: 140px;
                    padding: 10px 16px;
                    color: #03dac6;
                    font-weight: 500;
                    flex-shrink: 0;
                `;
                
                const valueCell = document.createElement('div');
                valueCell.textContent = value;
                valueCell.style.cssText = `
                    flex: 1;
                    padding: 10px 16px;
                    word-break: break-word;
                    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
                `;
                
                row.appendChild(keyCell);
                row.appendChild(valueCell);
                paramsTable.appendChild(row);
            }
            
            paramsSection.appendChild(paramsTable);
        } else {
            const valueEl = document.createElement('div');
            valueEl.textContent = call.parsedData;
            valueEl.style.cssText = `
                background: rgba(0,0,0,0.2);
                padding: 10px;
                border-radius: 6px;
                font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
                font-size: 14px;
            `;
            paramsSection.appendChild(valueEl);
        }
        
        content.appendChild(paramsSection);
    }
    
    // Raw data section if available
    if (call.rawData) {
        const rawDataSection = document.createElement('div');
        rawDataSection.style.cssText = `
            padding: 16px 20px;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        `;
        
        const rawDataLabel = document.createElement('div');
        rawDataLabel.textContent = 'Raw Call Data';
        rawDataLabel.style.cssText = 'font-size: 14px; color: #888; margin-bottom: 6px;';
        
        const rawDataValue = document.createElement('pre');
        rawDataValue.textContent = call.rawData;
        rawDataValue.style.cssText = `
            font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
            background: rgba(0,0,0,0.2);
            padding: 10px;
            border-radius: 6px;
            overflow-x: auto;
            font-size: 12px;
            margin: 0;
            max-height: 120px;
            overflow-y: auto;
        `;
        
        // Add copy button for raw data
        const copyBtn = document.createElement('button');
        copyBtn.textContent = 'Copy Raw Data';
        copyBtn.style.cssText = `
            background: #333;
            border: none;
            color: #fff;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
            margin-top: 8px;
        `;
        
        copyBtn.addEventListener('click', () => {
            navigator.clipboard.writeText(call.rawData);
            copyBtn.textContent = 'Copied!';
            setTimeout(() => { copyBtn.textContent = 'Copy Raw Data'; }, 2000);
        });
        
        rawDataSection.appendChild(rawDataLabel);
        rawDataSection.appendChild(rawDataValue);
        rawDataSection.appendChild(copyBtn);
        content.appendChild(rawDataSection);
    }
    
    // Subcalls section if available
    if (call.subCalls && call.subCalls.length > 0) {
        const subcallsSection = document.createElement('div');
        subcallsSection.style.cssText = `
            margin-top: 24px;
            border-top: 1px solid rgba(255,255,255,0.1);
            padding-top: 20px;
        `;
        
        const subcallsLabel = document.createElement('div');
        subcallsLabel.textContent = `Subcalls (${call.subCalls.length})`;
        subcallsLabel.style.cssText = `
            margin-bottom: 16px;
            font-weight: 600;
            color: #bb86fc;
        `;
        
        subcallsSection.appendChild(subcallsLabel);
        
        // Add each subcall
        call.subCalls.forEach((subcall, index) => {
            const subcallContainer = document.createElement('div');
            subcallContainer.style.cssText = `
                margin-bottom: ${index < call.subCalls.length - 1 ? '24px' : '0'};
                background: rgba(0,0,0,0.15);
                border-radius: 8px;
                padding: 16px;
                border-left: 3px solid #bb86fc;
            `;
            
            const subcallTitle = document.createElement('div');
            subcallTitle.textContent = `Subcall #${index + 1}`;
            subcallTitle.style.cssText = 'font-size: 14px; color: #888; margin-bottom: 12px;';
            
            subcallContainer.appendChild(subcallTitle);
            subcallContainer.appendChild(createFunctionCall(subcall));
            subcallsSection.appendChild(subcallContainer);
        });
        
        content.appendChild(subcallsSection);
    }
    
    return callContainer;
}

/**
 * Creates a verification instructions section
 * @returns {HTMLElement} Verification instructions element
 */
function createVerificationInstructions() {
    const container = document.createElement('div');
    container.style.cssText = `
        background: #1e1e1e;
        border-radius: 12px;
        overflow: hidden;
        margin-bottom: 24px;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    `;
    
    // Header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: linear-gradient(90deg, rgba(3,218,198,0.3) 0%, rgba(3,218,198,0.1) 100%);
        border-bottom: 1px solid rgba(3,218,198,0.2);
    `;
    
    const title = document.createElement('h3');
    title.innerHTML = '🔍 VERIFICATION CHECKLIST';
    title.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #03dac6;
    `;
    
    header.appendChild(title);
    container.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    const instructions = [
        {
            title: "Check Transaction Details",
            desc: "Ensure all transaction details exactly match what you expect to approve."
        },
        {
            title: "Verify Hashes",
            desc: "Domain and message hashes should exactly match those shown on other devices."
        },
        {
            title: "Check Hardware Wallet",
            desc: "Your hardware wallet should display the exact same hashes for signing."
        },
        {
            title: "Ask for Help if Uncertain",
            desc: "If anything seems suspicious or unclear, seek assistance before approving."
        }
    ];
    
    instructions.forEach((item, index) => {
        const step = document.createElement('div');
        step.style.cssText = `
            display: flex;
            align-items: flex-start;
            margin-bottom: ${index < instructions.length - 1 ? '16px' : '0'};
        `;
        
        const number = document.createElement('div');
        number.textContent = index + 1;
        number.style.cssText = `
            background: rgba(3,218,198,0.15);
            color: #03dac6;
            width: 24px;
            height: 24px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            margin-right: 16px;
            flex-shrink: 0;
            margin-top: 2px;
        `;
        
        const textContainer = document.createElement('div');
        
        const stepTitle = document.createElement('div');
        stepTitle.textContent = item.title;
        stepTitle.style.cssText = `
            font-weight: 600;
            margin-bottom: 4px;
            color: #fff;
        `;
        
        const stepDesc = document.createElement('div');
        stepDesc.textContent = item.desc;
        stepDesc.style.cssText = `
            font-size: 14px;
            color: #aaa;
            line-height: 1.5;
        `;
        
        textContainer.appendChild(stepTitle);
        textContainer.appendChild(stepDesc);
        
        step.appendChild(number);
        step.appendChild(textContainer);
        content.appendChild(step);
    });
    
    container.appendChild(content);
    return container;
}

/**
 * Creates a nested transaction warning
 * @param {Object} nestedInfo - Information about the nested transaction
 * @returns {HTMLElement} Nested transaction warning element
 */
function createNestedTransactionWarning(nestedInfo) {
    const container = document.createElement('div');
    container.style.cssText = `
        background: linear-gradient(135deg, rgba(255,82,82,0.15) 0%, rgba(255,82,82,0.05) 100%);
        border-radius: 12px;
        overflow: hidden;
        margin-bottom: 24px;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    `;
    
    // Header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: rgba(255,82,82,0.2);
        border-bottom: 1px solid rgba(255,82,82,0.3);
        display: flex;
        align-items: center;
    `;
    
    const icon = document.createElement('span');
    icon.textContent = '⚠️';
    icon.style.cssText = 'font-size: 20px; margin-right: 12px;';
    
    const title = document.createElement('h3');
    title.textContent = 'NESTED TRANSACTION DETECTED';
    title.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #ff5252;
    `;
    
    header.appendChild(icon);
    header.appendChild(title);
    container.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    const warning = document.createElement('p');
    warning.textContent = 'This transaction approves the execution of another transaction. This could be used to execute malicious code. Please verify both transactions carefully.';
    warning.style.cssText = `
        margin: 0 0 20px 0;
        line-height: 1.6;
        color: #fff;
    `;
    
    content.appendChild(warning);
    
    // Nested transaction info
    const infoTitle = document.createElement('div');
    infoTitle.textContent = 'Nested Transaction Details';
    infoTitle.style.cssText = `
        font-weight: 600;
        margin-bottom: 12px;
        color: #fff;
    `;
    content.appendChild(infoTitle);
    
    // Add nested info
    const infoTable = document.createElement('div');
    infoTable.style.cssText = `
        background: rgba(0,0,0,0.2);
        border-radius: 8px;
        overflow: hidden;
        font-size: 14px;
    `;
    
    const info = [
        { key: 'Safe Address', value: nestedInfo.safe },
        { key: 'Nonce', value: nestedInfo.nonce },
        { key: 'Transaction Hash', value: nestedInfo.hash }
    ];
    
    info.forEach((item, index) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            ${index < info.length - 1 ? 'border-bottom: 1px solid rgba(255,255,255,0.05);' : ''}
        `;
        
        const keyCell = document.createElement('div');
        keyCell.textContent = item.key;
        keyCell.style.cssText = `
            width: 140px;
            padding: 10px 16px;
            color: #ff8a80;
            font-weight: 500;
            flex-shrink: 0;
        `;
        
        const valueCell = document.createElement('div');
        valueCell.style.cssText = `
            flex: 1;
            padding: 10px 16px;
            word-break: break-all;
            font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
        `;
        
        // Check for verified addresses
        if (typeof item.value === 'string' && item.value.includes(' ✅')) {
            const parts = item.value.split(' (');
            const address = document.createElement('span');
            address.textContent = parts[0];
            
            valueCell.appendChild(address);
            
            if (parts.length > 1) {
                const verified = document.createElement('span');
                verified.innerHTML = ` (<span style="color: #03dac6; font-weight: 500;">${parts[1]}</span>`;
                valueCell.appendChild(verified);
            }
        } else {
            valueCell.textContent = item.value;
        }
        
        row.appendChild(keyCell);
        row.appendChild(valueCell);
        infoTable.appendChild(row);
    });
    
    content.appendChild(infoTable);
    container.appendChild(content);
    
    return container;
}

/**
 * Creates a verification result view
 * @param {Object} result - The verification result data
 * @returns {HTMLElement} The complete verification result element
 */
function createVerificationResult(result) {
    const container = createResultContainer();
    container.appendChild(createVerificationHeader("Transaction Verification"));
    
    // Create tab navigation
    const tabs = createTabNavigation(["Transaction Summary", "Function Calls", "Security Hashes"]);
    container.appendChild(tabs.tabContainer);
    container.appendChild(tabs.contentSections[0].parentElement);
    
    // Tab 1: Transaction Summary
    const summaryTab = tabs.contentSections[0];
    
    // Add status banner or warning
    if (result.hasNestedTransaction) {
        summaryTab.appendChild(createStatusBanner(true));
    } else {
        summaryTab.appendChild(createStatusBanner(false));
    }
    
    // Add basic transaction info
    summaryTab.appendChild(createInfoCard("Transaction Details", result.basicInfo, { type: 'table' }));
    
    // Add nested transaction warning if needed
    if (result.hasNestedTransaction) {
        summaryTab.appendChild(createNestedTransactionWarning(result.nestedInfo));
    }
    
    // Add verification instructions
    summaryTab.appendChild(createVerificationInstructions());
    
    // Tab 2: Function Calls
    const callsTab = tabs.contentSections[1];
    
    if (result.callDetails) {
        // Display main function call
        callsTab.appendChild(createFunctionCall(result.callDetails));
    } else {
        // Display no function calls message
        const noCallsMsg = document.createElement('div');
        noCallsMsg.textContent = 'No function calls to display for this transaction.';
        noCallsMsg.style.cssText = `
            padding: 24px;
            text-align: center;
            color: #888;
            font-style: italic;
        `;
        callsTab.appendChild(noCallsMsg);
    }
    
    // Tab 3: Security Hashes
    const hashesTab = tabs.contentSections[2];
    
    // Add security explanation
    const hashExplanation = document.createElement('div');
    hashExplanation.style.cssText = `
        margin-bottom: 24px;
        background: rgba(0,0,0,0.2);
        border-radius: 12px;
        padding: 16px 20px;
    `;
    
    const hashTitle = document.createElement('h4');
    hashTitle.textContent = 'What are these hashes?';
    hashTitle.style.cssText = `
        margin: 0 0 8px 0;
        font-size: 16px;
        color: #fff;
    `;
    
    const hashDesc = document.createElement('p');
    hashDesc.textContent = 'These cryptographic hashes uniquely identify this transaction. Before approving, verify they match exactly what your hardware wallet displays.';
    hashDesc.style.cssText = `
        margin: 0;
        font-size: 14px;
        color: #aaa;
        line-height: 1.5;
    `;
    
    hashExplanation.appendChild(hashTitle);
    hashExplanation.appendChild(hashDesc);
    hashesTab.appendChild(hashExplanation);
    
    // Add hashes card
    hashesTab.appendChild(createInfoCard("Verification Hashes", result.hashes, { type: 'hash' }));
    
    return container;
}

/**
 * Creates a modern transaction verification view from first principles
 * @param {Object} result - The verification result data
 * @returns {HTMLElement} The complete transaction verification UI
 */
function createVerificationResult(result) {
    // First format the data from our result object
    const formattedData = formatResultForDisplay(result);
    
    // Create container with clean modern styling
    const container = document.createElement('div');
    container.style.cssText = `
        font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        background: #121212;
        color: #fff;
        border-radius: 16px;
        box-shadow: 0 10px 30px rgba(0,0,0,0.4);
        max-width: 960px;
        margin: 0 auto;
        overflow: hidden;
    `;
    
    // Create header with security status
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 32px;
        position: relative;
        background: ${formattedData.hasNestedTransaction ? 
            'linear-gradient(135deg, #470000 0%, #1a1a1a 100%)' : 
            'linear-gradient(135deg, #003a34 0%, #1a1a1a 100%)'};
    `;
    
    // Security indicator
    const securityIndicator = document.createElement('div');
    securityIndicator.className = 'security-indicator'; 
    securityIndicator.style.cssText = `
        display: flex;
        align-items: center;
        margin-bottom: 16px;
    `;
    
    const indicatorIcon = document.createElement('div');
    indicatorIcon.innerHTML = formattedData.hasNestedTransaction ? '⚠️' : '🔒';
    indicatorIcon.style.cssText = 'font-size: 24px; margin-right: 12px;';
    
    const indicatorText = document.createElement('div');
    indicatorText.style.cssText = 'font-weight: 600; font-size: 14px;';
    indicatorText.innerHTML = formattedData.hasNestedTransaction ? 
        '<span style="color: #ff5252;">REQUIRES CAREFUL REVIEW</span>' : 
        '<span style="color: #03dac6;">STANDARD TRANSACTION</span>';
    
    securityIndicator.appendChild(indicatorIcon);
    securityIndicator.appendChild(indicatorText);
    header.appendChild(securityIndicator);
    
    // Header title
    const title = document.createElement('h1');
    title.textContent = 'Transaction Verification';
    title.style.cssText = 'font-size: 28px; margin: 0 0 8px 0; font-weight: 700;';
    header.appendChild(title);
    
    // Security warning for nested transactions
    if (formattedData.hasNestedTransaction) {
        const nestedWarning = document.createElement('div');
        nestedWarning.style.cssText = `
            margin-top: 16px;
            background: rgba(255,82,82,0.15);
            border-left: 4px solid #ff5252;
            padding: 12px 16px;
            border-radius: 4px;
            font-size: 14px;
        `;
        nestedWarning.innerHTML = '<strong>Warning:</strong> This transaction approves another transaction. Verify both carefully.';
        header.appendChild(nestedWarning);
    }
    
    container.appendChild(header);
    
    // Create body content
    const body = document.createElement('div');
    body.style.cssText = `
        display: flex;
        flex-direction: column;
        gap: 24px;
        padding: 32px;
        background: #1a1a1a;
    `;
    
    // Transaction Card - includes core transaction information
    const txCard = createTransactionCard(formattedData.basicInfo);
    body.appendChild(txCard);
    
    // Function Call Card - includes what this transaction is doing
    if (formattedData.callDetails) {
        const functionCard = createFunctionCallCard(formattedData.callDetails);
        body.appendChild(functionCard);
    }
    
    // Nested Transaction Card - if this transaction contains another tx
    if (formattedData.hasNestedTransaction) {
        const nestedCard = createNestedTransactionCard(formattedData.nestedInfo);
        body.appendChild(nestedCard);
    }
    
    // Security Hash Card - important verification information
    const hashCard = createSecurityHashCard(formattedData.hashes);
    body.appendChild(hashCard);
    
    // Verification Checklist
    body.appendChild(createVerificationChecklist());
    
    container.appendChild(body);
    return container;
}

/**
 * Creates a card showing transaction details
 * @param {Object} txInfo - Basic transaction information
 * @returns {HTMLElement} Transaction card element
 */
function createTransactionCard(txInfo) {
    const card = document.createElement('div');
    card.style.cssText = `
        background: #242424;
        border-radius: 12px;
        overflow: hidden;
    `;
    
    // Card header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #2a2a2a;
        border-bottom: 1px solid #333;
    `;
    
    const title = document.createElement('h2');
    title.textContent = 'Transaction Details';
    title.style.cssText = `
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: #fff;
    `;
    
    header.appendChild(title);
    card.appendChild(header);
    
    // Card content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    // Create a table for transaction details
    const table = document.createElement('div');
    table.style.cssText = `
        width: 100%;
        border-spacing: 0;
        border-collapse: collapse;
        font-size: 14px;
    `;
    
    // Add each transaction detail as a row
    Object.entries(txInfo).forEach(([key, value], index) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            border-bottom: ${index < Object.keys(txInfo).length - 1 ? '1px solid #333' : 'none'};
        `;
        
        const keyCell = document.createElement('div');
        keyCell.style.cssText = `
            width: 120px;
            padding: 12px 16px;
            color: #03dac6;
            font-weight: 500;
            flex-shrink: 0;
        `;
        
        // Convert snake_case or camelCase to Title Case
        const formattedKey = key
            .replace(/([A-Z])/g, ' $1')
            .replace(/_/g, ' ')
            .replace(/^./, str => str.toUpperCase());
            
        keyCell.textContent = formattedKey;
        
        const valueCell = document.createElement('div');
        valueCell.style.cssText = `
            flex: 1;
            padding: 12px 16px;
            word-break: break-all;
        `;
        
        // Special formatting for contract addresses and known addresses
        if (typeof value === 'string' && value.includes(' ✓')) {
            const parts = value.split(' (');
            const addressEl = document.createElement('div');
            addressEl.textContent = parts[0];
            addressEl.style.cssText = 'font-family: monospace;';
            
            valueCell.appendChild(addressEl);
            
            if (parts.length > 1) {
                const verifiedEl = document.createElement('div');
                verifiedEl.innerHTML = `<span style="color: #03dac6; font-weight: 500;">✓ ${parts[1].replace('✅)', '')}</span>`;
                valueCell.appendChild(verifiedEl);
            }
        } else {
            valueCell.textContent = value;
        }
        
        row.appendChild(keyCell);
        row.appendChild(valueCell);
        table.appendChild(row);
    });
    
    content.appendChild(table);
    card.appendChild(content);
    return card;
}

/**
 * Creates a card showing function call information
 * @param {Object} callDetails - Details of the function call
 * @returns {HTMLElement} Function call card element
 */
function createFunctionCallCard(callDetails) {
    const card = document.createElement('div');
    card.style.cssText = `
        background: #242424;
        border-radius: 12px;
        overflow: hidden;
    `;
    
    // Card header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #2a2a2a;
        border-bottom: 1px solid #333;
        display: flex;
        align-items: center;
    `;
    
    const titleIcon = document.createElement('span');
    titleIcon.textContent = '🔷';
    titleIcon.style.cssText = 'font-size: 20px; margin-right: 12px;';
    
    const title = document.createElement('h2');
    title.textContent = 'Function Call';
    title.style.cssText = `
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: #fff;
    `;
    
    header.appendChild(titleIcon);
    header.appendChild(title);
    card.appendChild(header);
    
    // Card content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    // Function name
    const functionName = document.createElement('div');
    functionName.style.cssText = `
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 16px;
        color: #bb86fc;
        font-family: monospace;
    `;
    functionName.textContent = callDetails.function || 'Unknown Function';
    content.appendChild(functionName);
    
    // Target contract
    const targetSection = document.createElement('div');
    targetSection.style.cssText = 'margin-bottom: 20px;';
    
    const targetLabel = document.createElement('div');
    targetLabel.textContent = 'Target Contract';
    targetLabel.style.cssText = `
        color: #a0a0a0;
        font-size: 14px;
        margin-bottom: 8px;
        text-align: center;
    `;
    
    const targetValue = document.createElement('div');
    targetValue.style.cssText = `
        background: rgba(0,0,0,0.2);
        padding: 12px;
        border-radius: 8px;
        word-break: break-all;
        font-family: monospace;
        margin-bottom: 16px;
    `;
    
    // Check if the target has a verified name
    if (typeof callDetails.target === 'string' && callDetails.target.includes(' ✓')) {
        const parts = callDetails.target.split(' (');
        targetValue.textContent = parts[0];
        
        if (parts.length > 1) {
            const verifiedName = document.createElement('div');
            // Fix character: Replace ✅ with ✓ for consistent rendering
            verifiedName.innerHTML = `<span style="color: #03dac6; font-weight: 500;">✓ ${parts[1].replace('✅)', '')}</span>`;
            targetValue.appendChild(verifiedName);
        }
    } else {
        targetValue.textContent = callDetails.target || 'Unknown Contract';
    }
    
    targetSection.appendChild(targetLabel);
    targetSection.appendChild(targetValue);
    content.appendChild(targetSection);
    
    // Parameters
    if (callDetails.parsedData && Object.keys(callDetails.parsedData).length > 0) {
        const paramsSection = document.createElement('div');
        
        const paramsLabel = document.createElement('div');
        paramsLabel.textContent = 'Parameters';
        paramsLabel.style.cssText = `
            color: #a0a0a0;
            font-size: 14px;
            margin-bottom: 8px;
        `;
        
        paramsSection.appendChild(paramsLabel);
        
        const paramsTable = document.createElement('div');
        paramsTable.style.cssText = `
            background: #1a1a1a;
            border-radius: 6px;
            overflow: hidden;
        `;
        
        Object.entries(callDetails.parsedData).forEach(([key, value], index) => {
            const row = document.createElement('div');
            row.style.cssText = `
                display: flex;
                ${index < Object.keys(callDetails.parsedData).length - 1 ? 'border-bottom: 1px solid #333;' : ''}
                padding: 12px;
            `;
            
            const keyCell = document.createElement('div');
            keyCell.textContent = key;
            keyCell.style.cssText = `
                width: 120px;
                color: #bb86fc;
                font-weight: 500;
                font-family: monospace;
                margin-right: 16px;
                flex-shrink: 0;
            `;
            
            const valueCell = document.createElement('div');
            valueCell.textContent = value;
            valueCell.style.cssText = `
                flex: 1;
                word-break: break-all;
                font-family: monospace;
            `;
            
            row.appendChild(keyCell);
            row.appendChild(valueCell);
            paramsTable.appendChild(row);
        });
        
        paramsSection.appendChild(paramsTable);
        content.appendChild(paramsSection);
    }
    
    card.appendChild(content);
    return card;
}

/**
 * Creates a card showing nested transaction information
 * @param {Object} nestedInfo - Information about the nested transaction
 * @returns {HTMLElement} Nested transaction card element
 */
function createNestedTransactionCard(nestedInfo) {
    const card = document.createElement('div');
    card.style.cssText = `
        background: #2d2024;
        border-radius: 12px;
        overflow: hidden;
        border: 1px solid rgba(255,82,82,0.3);
    `;
    
    // Card header with warning
    const header = document.createElement('div');
    header.style.cssText = `
        background: linear-gradient(to right, #3a1518, #2d1a1a);
        padding: 16px 20px;
        border-bottom: 1px solid rgba(255,82,82,0.3);
        display: flex;
        align-items: center;
    `;
    
    const titleIcon = document.createElement('span');
    titleIcon.textContent = '⚠️';
    titleIcon.style.cssText = 'font-size: 20px; margin-right: 12px;';
    
    const title = document.createElement('h2');
    title.textContent = 'Nested Transaction';
    title.style.cssText = `
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: #ff5252;
    `;
    
    header.appendChild(titleIcon);
    header.appendChild(title);
    card.appendChild(header);
    
    // Card content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    // Warning message
    const warning = document.createElement('div');
    warning.style.cssText = `
        margin-bottom: 20px;
        line-height: 1.5;
    `;
    warning.textContent = 'This transaction contains another transaction. This is advanced functionality that could execute malicious code. Verify all details carefully.';
    content.appendChild(warning);
    
    // Nested transaction details title
    const detailsTitle = document.createElement('h3');
    detailsTitle.textContent = 'Nested Transaction Details';
    detailsTitle.style.cssText = `
        margin: 0 0 16px 0;
        font-size: 16px;
        font-weight: 600;
        color: #ff8a80;
    `;
    content.appendChild(detailsTitle);
    
    // Details table
    const table = document.createElement('div');
    table.style.cssText = `
        background: rgba(0,0,0,0.2);
        border-radius: 8px;
        overflow: hidden;
    `;
    
    // Add rows for each piece of nested transaction info
    const info = [
        { key: 'Safe Address', value: nestedInfo.safe },
        { key: 'Nonce', value: nestedInfo.nonce },
        { key: 'Hash', value: nestedInfo.hash }
    ];
    
    info.forEach(({ key, value }, index) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            ${index < info.length - 1 ? 'border-bottom: 1px solid rgba(255,255,255,0.1);' : ''}
            align-items: flex-start;
        `;
        
        const keyCell = document.createElement('div');
        keyCell.textContent = key;
        keyCell.style.cssText = `
            width: 120px;
            padding: 12px 16px;
            color: #ff8a80;
            font-weight: 500;
            flex-shrink: 0;
        `;
        
        const valueCell = document.createElement('div');
        valueCell.style.cssText = `
            flex: 1;
            padding: 12px 16px;
            word-break: break-all;
            font-family: monospace;
        `;
        
        // Special formatting for addresses with verification status
        if (typeof value === 'string' && value.includes(' ✓')) {
            const parts = value.split(' (');
            valueCell.textContent = parts[0];
            
            if (parts.length > 1) {
                const verifiedInfo = document.createElement('div');
                // Fix character: Replace ✅ with ✓ for consistent rendering
                verifiedInfo.innerHTML = `<span style="color: #03dac6; font-weight: 500;">✓ ${parts[1].replace('✅)', '')}</span>`;
                valueCell.appendChild(verifiedInfo);
            }
        } else {
            valueCell.textContent = value;
        }
        
        row.appendChild(keyCell);
        row.appendChild(valueCell);
        table.appendChild(row);
    });
    
    content.appendChild(table);
    
    // Nested function call if available
    if (nestedInfo.callDetails) {
        const nestedFunctionSection = document.createElement('div');
        nestedFunctionSection.style.cssText = 'margin-top: 20px;';
        
        const nestedFunctionTitle = document.createElement('h3');
        nestedFunctionTitle.textContent = 'Nested Function Call';
        nestedFunctionTitle.style.cssText = `
            margin: 0 0 16px 0;
            font-size: 16px;
            font-weight: 600;
            color: #ff8a80;
        `;
        
        nestedFunctionSection.appendChild(nestedFunctionTitle);
        
        // Function name
        const functionName = document.createElement('div');
        functionName.style.cssText = `
            background: rgba(0,0,0,0.2);
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 12px;
            color: #bb86fc;
            font-family: monospace;
            text-align: center;
        `;
        functionName.textContent = nestedInfo.callDetails.function || 'Unknown Function';
        
        nestedFunctionSection.appendChild(functionName);
        
        // Target contract if available
        if (nestedInfo.callDetails.target) {
            const targetLabel = document.createElement('div');
            targetLabel.textContent = 'Target:';
            targetLabel.style.cssText = `
                color: #a0a0a0;
                font-size: 14px;
                margin-bottom: 8px;
            `;
            
            const targetValue = document.createElement('div');
            targetValue.style.cssText = `
                background: rgba(0,0,0,0.2);
                padding: 12px;
                border-radius: 8px;
                word-break: break-all;
                font-family: monospace;
                margin-bottom: 16px;
            `;
            targetValue.textContent = nestedInfo.callDetails.target;
            
            nestedFunctionSection.appendChild(targetLabel);
            nestedFunctionSection.appendChild(targetValue);
        }
        
        content.appendChild(nestedFunctionSection);
    }
    
    card.appendChild(content);
    return card;
}

/**
 * Creates a card showing security hash information
 * @param {Object} hashes - Hash information for the transaction
 * @returns {HTMLElement} Security hash card element
 */
function createSecurityHashCard(hashes) {
    const card = document.createElement('div');
    card.style.cssText = `
        background: #242424;
        border-radius: 12px;
        overflow: hidden;
    `;
    
    // Card header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #2a2a2a;
        border-bottom: 1px solid #333;
        display: flex;
        align-items: center;
    `;
    
    const titleIcon = document.createElement('span');
    titleIcon.textContent = '🔐';
    titleIcon.style.cssText = 'font-size: 20px; margin-right: 12px;';
    
    const title = document.createElement('h2');
    title.textContent = 'Security Verification';
    title.style.cssText = `
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: #fff;
    `;
    
    header.appendChild(titleIcon);
    header.appendChild(title);
    card.appendChild(header);
    
    // Card content
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    // Hash explanation
    const explanation = document.createElement('div');
    explanation.style.cssText = `
        margin-bottom: 20px;
        line-height: 1.5;
        color: #a0a0a0;
        font-size: 14px;
    `;
    explanation.textContent = 'These cryptographic hashes uniquely identify this transaction. Verify they match exactly what your hardware wallet displays.';
    content.appendChild(explanation);
    
    // Add each hash
    Object.entries(hashes).forEach(([key, value]) => {
        const hashContainer = document.createElement('div');
        hashContainer.style.cssText = 'margin-bottom: 16px;';
        
        const hashLabel = document.createElement('div');
        hashLabel.textContent = key.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase());
        hashLabel.style.cssText = `
            font-weight: 500;
            margin-bottom: 8px;
            color: #bb86fc;
        `;
        
        const hashBox = document.createElement('div');
        hashBox.style.cssText = `
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            padding: 12px 16px;
            position: relative;
            display: flex;
            align-items: center;
            justify-content: space-between;
        `;
        
        const hashText = document.createElement('code');
        hashText.textContent = value;
        hashText.style.cssText = `
            font-family: monospace;
            color: #03dac6;
            font-size: 14px;
            word-break: break-all;
            flex: 1;
            margin-right: 12px;
        `;
        
        const copyBtn = document.createElement('button');
        copyBtn.textContent = 'Copy';
        copyBtn.style.cssText = `
            background: #333;
            border: none;
            color: #fff;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
            flex-shrink: 0;
        `;
        
        copyBtn.addEventListener('click', () => {
            navigator.clipboard.writeText(value);
            copyBtn.textContent = 'Copied!';
            setTimeout(() => { copyBtn.textContent = 'Copy'; }, 2000);
        });
        
        hashBox.appendChild(hashText);
        hashBox.appendChild(copyBtn);
        
        hashContainer.appendChild(hashLabel);
        hashContainer.appendChild(hashBox);
        content.appendChild(hashContainer);
    });
    
    card.appendChild(content);
    return card;
}

/**
 * Creates a verification checklist
 * @returns {HTMLElement} Verification checklist element
 */
function createVerificationChecklist() {
    const card = document.createElement('div');
    card.className = 'verification-checklist';
    card.style.cssText = `
        background: rgba(3,218,198,0.1);
        border-radius: 12px;
        overflow: hidden;
    `;
    
    // Card header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: rgba(3,218,198,0.2);
        border-bottom: 1px solid rgba(3,218,198,0.2);
        display: flex;
        align-items: center;
    `;
    
    const icon = document.createElement('span');
    icon.textContent = '🔍';
    icon.style.cssText = 'font-size: 20px; margin-right: 12px;';
    
    const title = document.createElement('h2');
    title.textContent = 'Verification Checklist';
    title.style.cssText = `
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: #03dac6;
    `;
    
    header.appendChild(icon);
    header.appendChild(title);
    card.appendChild(header);
    
    // Card content with verification steps
    const content = document.createElement('div');
    content.style.cssText = `padding: 20px;`;
    
    const steps = [
        {
            number: 1,
            title: "Check transaction details",
            desc: "Verify all transaction details exactly match what you expect to approve."
        },
        {
            number: 2,
            title: "Verify contract addresses",
            desc: "Ensure all contract addresses are verified and match expected contracts."
        },
        {
            number: 3,
            title: "Check security hashes",
            desc: "The hashes shown on your hardware wallet must exactly match those shown here."
        },
        {
            number: 4,
            title: "Exercise caution with nested transactions",
            desc: "Nested transactions can be complex and require careful review."
        }
    ];
    
    steps.forEach((step, index) => {
        const stepEl = document.createElement('div');
        stepEl.style.cssText = `
            display: flex;
            align-items: flex-start;
            margin-bottom: ${index < steps.length - 1 ? '16px' : '0'};
        `;
        
        const stepNumber = document.createElement('div');
        stepNumber.textContent = step.number;
        stepNumber.style.cssText = `
            background: rgba(3,218,198,0.2);
            color: #03dac6;
            width: 28px;
            height: 28px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            margin-right: 16px;
            flex-shrink: 0;
        `;
        
        const stepContent = document.createElement('div');
        
        const stepTitle = document.createElement('div');
        stepTitle.textContent = step.title;
        stepTitle.style.cssText = `
            font-weight: 600;
            margin-bottom: 4px;
            color: #fff;
        `;
        
        const stepDesc = document.createElement('div');
        stepDesc.textContent = step.desc;
        stepDesc.style.cssText = `
            color: #aaa;
            font-size: 14px;
            line-height: 1.5;
        `;
        
        stepContent.appendChild(stepTitle);
        stepContent.appendChild(stepDesc);
        
        stepEl.appendChild(stepNumber);
        stepEl.appendChild(stepContent);
        content.appendChild(stepEl);
    });
    
    card.appendChild(content);
    return card;
}